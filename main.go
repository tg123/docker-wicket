package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/docker/docker/pkg/mflag"
	"github.com/gocraft/web"
	"github.com/rakyll/globalconf"

	"github.com/tg123/docker-wicket/acl"
	"github.com/tg123/docker-wicket/index"

	"github.com/tg123/docker-wicket/handler"
	"github.com/tg123/docker-wicket/handler/v1"
	"github.com/tg123/docker-wicket/handler/v2"
)

// parse conf from env and args
func parseConf() {

	// let mflag parse first
	mflag.Parse()

	// using gconf parse env
	gconf, err := globalconf.NewWithOptions(&globalconf.Options{
		EnvPrefix: "WICKET_",
	})

	if err != nil {
		log.Fatalf("error parsing config file: %v", err)
	}

	fs := flag.NewFlagSet("", flag.ContinueOnError)

	mflag.VisitAll(func(f *mflag.Flag) {
		for _, n := range f.Names {
			if len(n) < 2 {
				continue
			}

			n = strings.TrimPrefix(n, "-")
			fs.Var(f.Value, n, f.Usage)
		}
	})

	gconf.ParseSet("", fs)
}

// TODO mmore log
func main() {

	var ListenAddr string
	var Port uint

	tokenAuth := &handler.TokenAuth{}

	// http
	mflag.StringVar(&ListenAddr, []string{"l", "-addr"}, "0.0.0.0", "Listening Address")
	mflag.UintVar(&Port, []string{"p", "-port"}, 9999, "Listening Port")

	// acl
	var aclDriverName string
	mflag.StringVar(&aclDriverName, []string{"-acl_driver"}, "", "ACL Driver for Docker Wicket")

	// token for v1 and v2
	mflag.StringVar(&tokenAuth.Issuer, []string{"-issuer"}, "docker-wicket", "Issuer of the token, MUST be same as what in registy2")
	mflag.StringVar(&tokenAuth.Service, []string{"-service"}, "registry", "Service of the token")
	mflag.Int64Var(&tokenAuth.Expiration, []string{"-expiration"}, 600, "how long the token can be treated as valid. (sec)")

	// cert and key for token
	var certPath string
	var certKeyPath string
	mflag.StringVar(&certPath, []string{"-cert"}, "", "Token certificate file path, MUST be in the bundle of registy2")
	mflag.StringVar(&certKeyPath, []string{"-key"}, "", "Key file path to token certificate")

	// v1 only
	var indexDriverName string
	var v1Endpoint string
	mflag.StringVar(&v1Endpoint, []string{"-v1_endpoint"}, "", "Endpoint of registry1")
	mflag.StringVar(&indexDriverName, []string{"-v1_index_driver"}, "", "Index driver of registry1")

	parseConf()

	err := tokenAuth.LoadCertAndKey(certPath, certKeyPath)
	if err != nil {
		log.Fatalf("Cannot load cert: %v", err)
	}

	acldriver, err := acl.Load(aclDriverName)
	if err != nil {
		log.Fatalf("Cannot load ACL Driver: %v", err)
	}

	indexdriver, err := index.Load(indexDriverName)
	if err != nil {
		log.Fatalf("Cannot load index Driver: %v", err)
	}

	router := web.New(handler.ShareWebContext{}).
		Middleware(web.LoggerMiddleware)

	v1.InstallHandler(router, &v1.RunningContext{
		RunningContext: handler.RunningContext{
			TokenAuth: tokenAuth,
			Acl:       acldriver,
		},
		// spec
		Endpoints: v1Endpoint,
		Index:     indexdriver,
	})

	v2.InstallHandler(router, &v2.RunningContext{
		RunningContext: handler.RunningContext{
			Acl:       acldriver,
			TokenAuth: tokenAuth,
		},
	})

	log.Printf("Docker wicket @ %v:%v", ListenAddr, Port)

	log.Fatal(http.ListenAndServe(fmt.Sprintf("%v:%v", ListenAddr, Port), router))
}
