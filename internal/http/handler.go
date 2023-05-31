package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Deezy102/yc-key-manager/pkg/yandex"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/kms/v1"
	yc "github.com/yandex-cloud/go-sdk"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

const (
	timeout = 15 * time.Second
)

type Result struct {
	Body  interface{}
	Error error
}

type KeyHandler struct {
	IAMconf  *yandex.Config
	SDK      *yc.SDK
	Logger   *logrus.Logger
	FolderId string
	KeyName  string
}

func New(iconf *yandex.Config, l *logrus.Logger, fid, kn string) (*KeyHandler, error) {
	iam, err := yandex.NewIAM(iconf)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	sdk, err := yc.Build(ctx, yc.Config{
		Credentials: yc.NewIAMTokenCredentials(iam.Value()),
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to create sdk instance")
	}
	h := &KeyHandler{
		SDK:      sdk,
		Logger:   l,
		FolderId: fid,
		KeyName:  kn,
	}
	return h, nil
}

func (h *KeyHandler) SendKey(w http.ResponseWriter, r *http.Request) {
	ls, err := h.SDK.KMS().SymmetricKey().List(
		context.Background(),
		&kms.ListSymmetricKeysRequest{
			FolderId: h.FolderId,
		},
	)
	if err != nil {
		h.Logger.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Result{Error: err}) // nolint
		return
	}
	for _, k := range ls.Keys {
		if k.Name == h.KeyName {
			// fmt.Println(k)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Result{Body: k}) // nolint
			return
		}
	}
	ops, err := h.SDK.KMS().SymmetricKey().Create(
		context.Background(),
		&kms.CreateSymmetricKeyRequest{
			FolderId:           h.FolderId, // из конфига
			Name:               h.KeyName,
			Description:        "created by KM",
			DefaultAlgorithm:   kms.SymmetricAlgorithm_AES_256,
			RotationPeriod:     durationpb.New(time.Duration(25 * time.Hour)),
			DeletionProtection: false,
		},
	)
	if err != nil {
		h.Logger.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Result{Error: err}) // nolint
		return
	}
	var id string
	if ops.Done {
		resp := ops.GetResponse().GetValue()
		var key kms.SymmetricKey

		err := proto.Unmarshal(resp, &key)
		if err != nil {
			h.Logger.Error(err)
		}
		id = key.GetId()

	} else {
		h.Logger.Println("bad response from yc: can not create sym-key")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Result{Error: fmt.Errorf("%s", "bad response from yc: can not create sym-key")}) // nolint
		return
	}

	key, err := h.SDK.KMS().SymmetricKey().Get(
		context.Background(),
		&kms.GetSymmetricKeyRequest{
			KeyId: id,
		},
	)
	if err != nil {
		h.Logger.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Result{Error: fmt.Errorf("%s", "bad request to yc: cant get sym-key")}) // nolint
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Result{Body: key}) // nolint
}
