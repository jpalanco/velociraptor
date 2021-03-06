package flows

import (
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	errors "github.com/pkg/errors"
	"path"
	"strings"
	actions_proto "www.velocidex.com/golang/velociraptor/actions/proto"
	config "www.velocidex.com/golang/velociraptor/config"
	crypto_proto "www.velocidex.com/golang/velociraptor/crypto/proto"
	datastore "www.velocidex.com/golang/velociraptor/datastore"
	flows_proto "www.velocidex.com/golang/velociraptor/flows/proto"
	"www.velocidex.com/golang/velociraptor/responder"
	debug "www.velocidex.com/golang/velociraptor/testing"
	urns "www.velocidex.com/golang/velociraptor/urns"
	utils "www.velocidex.com/golang/velociraptor/utils"
	vql_subsystem "www.velocidex.com/golang/velociraptor/vql"
)

const (
	_VFSListDirectory_ListDir          uint64 = 1
	_VFSListDirectory_RecursiveListDir uint64 = 2
)

type VFSListDirectory struct {
	state *flows_proto.VFSListRequestState
	rows  []map[string]interface{}
}

func (self *VFSListDirectory) Load(
	config_obj *config.Config,
	flow_obj *AFF4FlowObject) error {
	message := flow_obj.GetState()
	if message == nil {
		message = &flows_proto.VFSListRequestState{
			Current: &actions_proto.VQLResponse{},
		}
	}
	self.state = message.(*flows_proto.VFSListRequestState)
	return json.Unmarshal([]byte(self.state.Current.Response), &self.rows)
}

func (self *VFSListDirectory) Save(
	config_obj *config.Config,
	flow_obj *AFF4FlowObject) error {
	s, err := json.Marshal(self.rows)
	if err != nil {
		return errors.WithStack(err)
	}
	self.state.Current.Response = string(s)
	self.state.Current.TotalRows = uint64(len(self.rows))
	flow_obj.SetState(self.state)
	return nil
}

func (self *VFSListDirectory) Start(
	config_obj *config.Config,
	flow_obj *AFF4FlowObject,
	args proto.Message) error {

	vfs_args, ok := args.(*flows_proto.VFSListRequest)
	if !ok {
		return errors.New("Expected args of type VQLCollectorArgs")
	}

	glob := "'/*'"
	next_state := _VFSListDirectory_ListDir
	if vfs_args.RecursionDepth > 0 {
		glob = fmt.Sprintf("'/**%d'", vfs_args.RecursionDepth)
		next_state = _VFSListDirectory_RecursiveListDir
	}

	vql_collector_args := &actions_proto.VQLCollectorArgs{
		// Injecting the path in the environment avoids the
		// need to escape it within the query itself and it is
		// therefore more robust.
		Env: []*actions_proto.VQLEnv{
			{Key: "path", Value: path.Join("/", vfs_args.VfsPath)},
		},
		Query: []*actions_proto.VQLRequest{
			{
				Name: vfs_args.VfsPath,
				VQL: "SELECT FullPath as _FullPath, " +
					"Name, Size, Mode.String AS Mode, " +
					"timestamp(epoch=Mtime.Sec) as mtime, " +
					"timestamp(epoch=Atime.Sec) as atime, " +
					"timestamp(epoch=Ctime.Sec) as ctime " +
					"from glob(globs=path + " + glob + ")",
			},
		},
		MaxRow: uint64(10000),
	}

	err := QueueMessageForClient(
		config_obj, flow_obj,
		"VQLClientAction",
		vql_collector_args, next_state)
	if err != nil {
		return err
	}

	self.state = &flows_proto.VFSListRequestState{
		Current: &actions_proto.VQLResponse{
			Query: vql_collector_args.Query[0],
		},
	}

	return nil
}

func (self *VFSListDirectory) ProcessMessage(
	config_obj *config.Config,
	flow_obj *AFF4FlowObject,
	message *crypto_proto.GrrMessage) error {

	switch message.RequestId {
	case _VFSListDirectory_ListDir:
		return self.processSingleDirectoryListing(
			config_obj, flow_obj, message)

	case _VFSListDirectory_RecursiveListDir:
		return self.processRecursiveDirectoryListing(
			config_obj, flow_obj, message)
	}

	return nil
}

func (self *VFSListDirectory) processSingleDirectoryListing(
	config_obj *config.Config,
	flow_obj *AFF4FlowObject,
	message *crypto_proto.GrrMessage) error {

	var tmp_args ptypes.DynamicAny
	err := ptypes.UnmarshalAny(flow_obj.RunnerArgs.Args, &tmp_args)
	if err != nil {
		return errors.WithStack(err)
	}

	vfs_args := tmp_args.Message.(*flows_proto.VFSListRequest)
	err = flow_obj.FailIfError(config_obj, message)
	if err != nil {
		return err
	}

	db, err := datastore.GetDB(config_obj)
	if err != nil {
		return err
	}

	if flow_obj.IsRequestComplete(message) {
		return flow_obj.Complete(config_obj)
	}

	response, ok := responder.ExtractGrrMessagePayload(
		message).(*actions_proto.VQLResponse)
	if !ok {
		return errors.New("Unexpected response type " + message.ArgsRdfName)
	}

	urn := urns.BuildURN(
		flow_obj.RunnerArgs.ClientId, "vfs",
		vfs_args.VfsPath)

	return db.SetSubject(config_obj, urn, response)
}

// Handle recursive VFS traversal. NOTE: This algorithm relies on the
// fact that the recursive glob (** wildcard) returns files in breadth
// first order. We keep track of the previous directory and add rows
// to the current collection as long as they belong to the current
// directory. When we see a file which belongs in another directory,
// we can flush the current collection and start a new one.
func (self *VFSListDirectory) processRecursiveDirectoryListing(
	config_obj *config.Config,
	flow_obj *AFF4FlowObject,
	message *crypto_proto.GrrMessage) error {

	if flow_obj.IsRequestComplete(message) {
		err := self.flush_state(config_obj, flow_obj)
		if err != nil {
			return err
		}
		return flow_obj.Complete(config_obj)
	}

	vql_response, ok := responder.ExtractGrrMessagePayload(
		message).(*actions_proto.VQLResponse)
	if !ok {
		return errors.New("Unexpected response type " + message.ArgsRdfName)
	}

	// Inspect each row and check if it belongs to the current
	// collection.
	var rows []map[string]interface{}
	err := json.Unmarshal([]byte(vql_response.Response), &rows)
	if err != nil {
		return errors.WithStack(err)
	}

	for _, row := range rows {
		full_path, ok := (row["_FullPath"]).(string)
		if ok {
			full_path = utils.Normalize_windows_path(full_path)
			// This row does not belong in the current
			// collection - flush the collection and start
			// a new one.
			if path.Dir(full_path) != self.state.VfsPath ||
				// Do not let our memory footprint
				// grow without bounds.
				len(self.rows) > 10000 {
				// VfsPath == "" represents the first
				// collection before the first row is
				// processed.
				if self.state.VfsPath != "" {
					err := self.flush_state(config_obj, flow_obj)
					if err != nil {
						return err
					}
				}
				self.state.VfsPath = path.Dir(full_path)
				self.state.Current = &actions_proto.VQLResponse{
					Query:     vql_response.Query,
					Columns:   vql_response.Columns,
					Timestamp: vql_response.Timestamp,
				}
			}
			self.rows = append(self.rows, row)
		}
	}

	return nil
}

// Flush the current state into the database and clear it for the next directory.
func (self *VFSListDirectory) flush_state(
	config_obj *config.Config,
	flow_obj *AFF4FlowObject) error {
	// Save will serialize the rows into self.state.Current
	err := self.Save(config_obj, flow_obj)
	if err != nil {
		return err
	}
	self.rows = nil

	urn := urns.BuildURN(
		flow_obj.RunnerArgs.ClientId, "vfs",
		self.state.VfsPath)

	db, err := datastore.GetDB(config_obj)
	if err != nil {
		return err
	}
	return db.SetSubject(config_obj, urn, self.state.Current)
}

type VFSDownloadFile struct {
	*BaseFlow
}

func (self *VFSDownloadFile) Start(
	config_obj *config.Config,
	flow_obj *AFF4FlowObject,
	args proto.Message) error {
	vfs_download_args, ok := args.(*flows_proto.VFSDownloadFileRequest)
	if !ok {
		return errors.New("Expected args of type VFSDownloadFileRequest")
	}

	request := &actions_proto.VQLCollectorArgs{}
	paths := []string{}
	for _, file_path := range vfs_download_args.VfsPath {
		file_path = path.Clean(file_path)
		if strings.HasPrefix(file_path, "/fs/") {
			local_path := strings.TrimPrefix(file_path, "/fs")
			path_var := fmt.Sprintf("Path%d", len(paths))
			paths = append(paths, path_var)
			request.Env = append(request.Env, &actions_proto.VQLEnv{
				Key:   path_var,
				Value: local_path,
			})
		} else {
			return errors.New(fmt.Sprintf(
				"Fetching VFS path %s not supported "+
					"(shoud start with '/fs/').", file_path))
		}
	}

	if len(paths) == 0 {
		return errors.New("Must specify some paths.")
	}

	request.Query = append(request.Query, &actions_proto.VQLRequest{
		Name: "Upload files",
		VQL: fmt.Sprintf("SELECT Path, Size, Error FROM upload(files=[%s])",
			strings.Join(paths, ", ")),
	})

	debug.Debug(request)
	err := QueueMessageForClient(
		config_obj, flow_obj,
		"VQLClientAction",
		request, processVQLResponses)
	if err != nil {
		return err
	}

	return nil
}

func (self *VFSDownloadFile) ProcessMessage(
	config_obj *config.Config,
	flow_obj *AFF4FlowObject,
	message *crypto_proto.GrrMessage) error {
	err := flow_obj.FailIfError(config_obj, message)
	if err != nil {
		return err
	}

	switch message.RequestId {
	case processVQLResponses:
		if flow_obj.IsRequestComplete(message) {
			return flow_obj.Complete(config_obj)
		}

		err = StoreResultInFlow(config_obj, flow_obj, message)
		if err != nil {
			return err
		}

		// Receive any file upload the client sent.
	case vql_subsystem.TransferWellKnownFlowId:
		return appendDataToFile(
			config_obj, flow_obj,
			path.Join(flow_obj.RunnerArgs.ClientId, "vfs_files"),
			message)
	}
	return nil
}

func init() {
	impl := VFSListDirectory{}
	default_args, _ := ptypes.MarshalAny(&flows_proto.VFSListRequest{})
	desc := &flows_proto.FlowDescriptor{
		Name:         "VFSListDirectory",
		FriendlyName: "List VFS Directory",
		Category:     "Collectors",
		Doc:          "List a single directory in the client's filesystem.",
		ArgsType:     "VFSListRequest",
		DefaultArgs:  default_args,
	}
	RegisterImplementation(desc, &impl)

	{
		impl := VFSDownloadFile{}
		default_args, _ := ptypes.MarshalAny(&flows_proto.VFSDownloadFileRequest{})
		desc := &flows_proto.FlowDescriptor{
			Name:         "VFSDownloadFile",
			FriendlyName: "Download VFS Files",
			Category:     "Collectors",
			Doc:          "Download files into the client's virtual filesystem.",
			ArgsType:     "VFSDownloadFileRequest",
			DefaultArgs:  default_args,
		}
		RegisterImplementation(desc, &impl)
	}
}
