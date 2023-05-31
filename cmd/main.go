package main

import (
	"net/http"
	"os"

	"github.com/Deezy102/yc-key-manager/internal/config"
	handlers "github.com/Deezy102/yc-key-manager/internal/http"
	"github.com/Deezy102/yc-key-manager/pkg/yandex"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	err := config.Load("./config")
	if err != nil {
		log.Fatal(err)
	}
	l := log.New()
	l.SetOutput(os.Stdout)
	l.SetReportCaller(true)
	l.SetFormatter(&log.JSONFormatter{
		FieldMap: log.FieldMap{
			log.FieldKeyTime:  "timestamp",
			log.FieldKeyLevel: "level",
			log.FieldKeyMsg:   "message",
			log.FieldKeyFunc:  "caller",
		},
	})
	// l.AddHook(logger.NewHook(viper.GetString("logging_url")))

	h, err := handlers.New(
		&yandex.Config{
			ServiceAccountID: viper.GetString("service_account_id"),
			KeyFile:          viper.GetString("key_file"),
			KeyID:            viper.GetString("key_id"),
		},
		l,
		viper.GetString("folder_id"),
		viper.GetString("key_name"),
	)
	if err != nil {
		l.Fatal(err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/key/", h.SendKey).Methods("GET")

	addr := viper.GetString("address")
	l.Println("start serving on ", addr)
	err = http.ListenAndServe(addr, r) //nolint:gosec//no need of timout here
	if err != nil {
		l.Fatal(err)
	}
}
