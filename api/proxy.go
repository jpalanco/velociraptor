package api

import (
	"fmt"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"net/http"
	api_proto "www.velocidex.com/golang/velociraptor/api/proto"
	"www.velocidex.com/golang/velociraptor/config"
	"www.velocidex.com/golang/velociraptor/logging"
)

func StartHTTPProxy(config_obj *config.Config) error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := http.NewServeMux()
	h, err := GetAPIHandler(ctx, config_obj)
	if err != nil {
		return err
	}
	mux.Handle("/api/", checkUserCredentialsHandler(config_obj, h))
	mux.Handle("/api/v1/download/", checkUserCredentialsHandler(
		config_obj, flowResultDownloadHandler(config_obj)))
	mux.Handle("/api/v1/DownloadHuntResults", checkUserCredentialsHandler(
		config_obj, huntResultDownloadHandler(config_obj)))
	mux.Handle("/api/v1/DownloadVFSFile/", checkUserCredentialsHandler(
		config_obj, vfsFileDownloadHandler(config_obj)))

	// Assets etc do not need auth.
	install_mux(config_obj, mux)

	h, err = GetTemplateHandler(config_obj, "/static/templates/app.html")
	if err != nil {
		return err
	}
	mux.Handle("/app.html", checkUserCredentialsHandler(config_obj, h))

	h, err = GetTemplateHandler(config_obj, "/static/templates/index.html")
	if err != nil {
		return err
	}
	mux.Handle("/", h)

	listenAddr := fmt.Sprintf("%s:%d",
		config_obj.GUI.BindAddress,
		config_obj.GUI.BindPort)

	logging.NewLogger(config_obj).Info(
		"GUI is ready to handle requests at %s", listenAddr)
	return http.ListenAndServe(listenAddr,
		logging.GetLoggingHandler(config_obj)(mux))
}

type _templateArgs struct {
	Timestamp  int64
	Heading    string
	Help_url   string
	Report_url string
	Version    string
}

func GetAPIHandler(
	ctx context.Context,
	config_obj *config.Config) (http.Handler, error) {

	// We need to tell when someone uses HEAD method on our grpc
	// proxy so we need to pass this information from the request
	// to the gRPC server using the gRPC metadata.
	grpc_proxy_mux := runtime.NewServeMux(
		runtime.WithMetadata(
			func(ctx context.Context, req *http.Request) metadata.MD {
				md := map[string]string{
					"METHOD": req.Method,
				}
				username, ok := req.Context().Value("USER").(string)
				if ok {
					md["USER"] = username
				}

				return metadata.New(md)
			}),
	)
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := api_proto.RegisterAPIHandlerFromEndpoint(
		ctx, grpc_proxy_mux,
		fmt.Sprintf("%s:%d",
			config_obj.API.BindAddress,
			config_obj.API.BindPort),
		opts)
	if err != nil {
		return nil, err
	}

	reverse_proxy_mux := http.NewServeMux()
	reverse_proxy_mux.Handle("/api/v1/", grpc_proxy_mux)

	return reverse_proxy_mux, nil
}
