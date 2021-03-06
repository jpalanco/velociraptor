package flows

import (
	"github.com/golang/protobuf/proto"
	config "www.velocidex.com/golang/velociraptor/config"
	datastore "www.velocidex.com/golang/velociraptor/datastore"
)

func QueueMessageForClient(
	config_obj *config.Config,
	flow_obj *AFF4FlowObject,
	client_action_name string,
	message proto.Message,
	next_state uint64) error {
	db, err := datastore.GetDB(config_obj)
	if err != nil {
		return err
	}

	return db.QueueMessageForClient(
		config_obj, flow_obj.RunnerArgs.ClientId,
		flow_obj.Urn,
		client_action_name, message, next_state)
}
