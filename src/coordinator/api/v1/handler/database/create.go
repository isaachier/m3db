// Copyright (c) 2018 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package database

import (
	"errors"
	"net/http"
	"time"

	clusterclient "github.com/m3db/m3cluster/client"
	"github.com/m3db/m3cluster/generated/proto/placementpb"
	"github.com/m3db/m3db/src/cmd/services/m3coordinator/config"
	"github.com/m3db/m3db/src/coordinator/api/v1/handler"
	"github.com/m3db/m3db/src/coordinator/api/v1/handler/namespace"
	"github.com/m3db/m3db/src/coordinator/api/v1/handler/placement"
	"github.com/m3db/m3db/src/coordinator/generated/proto/admin"
	"github.com/m3db/m3db/src/coordinator/util"
	"github.com/m3db/m3db/src/coordinator/util/logging"
	protonamespace "github.com/m3db/m3db/src/dbnode/generated/proto/namespace"
	dbnamespace "github.com/m3db/m3db/src/dbnode/storage/namespace"

	"github.com/golang/protobuf/jsonpb"
	"go.uber.org/zap"
)

const (
	// CreateURL is the url for the database create handler.
	CreateURL = "/api/v1/database/create"

	// CreateHTTPMethod is the HTTP method used with this resource.
	CreateHTTPMethod = "POST"

	dbTypeLocal = "local"
)

var (
	errMissingRequiredField = errors.New("all attributes must be set")
	errInvalidDBType        = errors.New("invalid database type")
)

type createHandler struct {
	placementInitHandler   *placement.InitHandler
	namespaceAddHandler    *namespace.AddHandler
	namespaceDeleteHandler *namespace.DeleteHandler
}

// NewCreateHandler returns a new instance of a database create handler.
func NewCreateHandler(client clusterclient.Client, cfg config.Configuration) http.Handler {
	return &createHandler{
		placementInitHandler:   placement.NewInitHandler(client, cfg),
		namespaceAddHandler:    namespace.NewAddHandler(client),
		namespaceDeleteHandler: namespace.NewDeleteHandler(client),
	}
}

func (h *createHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := logging.WithContext(ctx)

	namespaceRequest, placementRequest, rErr := h.parseRequest(r)
	if rErr != nil {
		logger.Error("unable to parse request", zap.Any("error", rErr))
		handler.Error(w, rErr.Error(), rErr.Code())
		return
	}

	nsRegistry, err := h.namespaceAddHandler.Add(namespaceRequest)
	if err != nil {
		logger.Error("unable to add namespace", zap.Any("error", err))
		handler.Error(w, err, http.StatusInternalServerError)
		return
	}

	initPlacement, err := h.placementInitHandler.Init(placementRequest)
	if err != nil {
		// Attempt to delete the namespace that was just created to maintain idempotency
		err = h.namespaceDeleteHandler.Delete(namespaceRequest.Name)
		if err != nil {
			logger.Error("unable to delete namespace we just added", zap.Any("error", err))
			handler.Error(w, err, http.StatusInternalServerError)
			return
		}

		logger.Error("unable to add namespace", zap.Any("error", err))
		handler.Error(w, err, http.StatusInternalServerError)
		return
	}

	placementProto, err := initPlacement.Proto()
	if err != nil {
		logger.Error("unable to get placement protobuf", zap.Any("error", err))
		handler.Error(w, err, http.StatusInternalServerError)
		return
	}

	resp := &admin.DatabaseCreateResponse{
		Namespace: &admin.NamespaceGetResponse{
			Registry: &nsRegistry,
		},
		Placement: &admin.PlacementGetResponse{
			Placement: placementProto,
		},
	}

	handler.WriteProtoMsgJSONResponse(w, resp, logger)
}

func (h *createHandler) parseRequest(r *http.Request) (*admin.NamespaceAddRequest, *admin.PlacementInitRequest, *handler.ParseError) {
	dbCreateReq := new(admin.DatabaseCreateRequest)
	if err := jsonpb.Unmarshal(r.Body, dbCreateReq); err != nil {
		return nil, nil, handler.NewParseError(err, http.StatusBadRequest)
	}

	if dbCreateReq.Type != dbTypeLocal {
		return nil, nil, handler.NewParseError(errInvalidDBType, http.StatusBadRequest)
	}

	if util.HasEmptyString(dbCreateReq.NamespaceName, dbCreateReq.Type) {
		return nil, nil, handler.NewParseError(errMissingRequiredField, http.StatusBadRequest)
	}

	namespaceAddRequest := defaultedNamespaceAddRequest(dbCreateReq)
	placementInitRequest := defaultedPlacementInitRequest(dbCreateReq)

	return namespaceAddRequest, placementInitRequest, nil
}

func defaultedNamespaceAddRequest(r *admin.DatabaseCreateRequest) *admin.NamespaceAddRequest {
	options := dbnamespace.NewOptions()

	switch r.Type {
	case dbTypeLocal:
		options.SetBootstrapEnabled(true)
		options.SetFlushEnabled(true)
		options.SetWritesToCommitLog(true)
		options.SetCleanupEnabled(true)
		options.SetRepairEnabled(false)
		options.SetSnapshotEnabled(false)

		// ExpectedSeriesDatapointsPerHour takes precedence over RetentionPeriodNanos
		if r.ExpectedSeriesDatapointsPerHour > 0 {
			// We want around 720 datapoints per block, so:
			// (720 / r.ExpectedSeriesDatapointsPerHour) * 60 * 60000000000
			options.RetentionOptions().SetRetentionPeriod(time.Duration(2592000000000000 / r.ExpectedSeriesDatapointsPerHour))
		} else {
			options.RetentionOptions().SetRetentionPeriod(time.Duration(r.RetentionPeriodNanos))
		}

		retentionPeriod := options.RetentionOptions().RetentionPeriod()
		if retentionPeriod <= 0 {
			options.RetentionOptions().SetRetentionPeriod(48 * time.Hour)
		}

		if retentionPeriod <= 48*time.Hour {
			options.RetentionOptions().SetBlockSize(2 * time.Hour)
		} else if retentionPeriod <= 336*time.Hour { // 14 * 24h
			options.RetentionOptions().SetBlockSize(4 * time.Hour)
		} else if retentionPeriod <= 720*time.Hour { // 30 * 24h
			options.RetentionOptions().SetBlockSize(12 * time.Hour)
		} else {
			options.RetentionOptions().SetBlockSize(24 * time.Hour)
		}
		options.RetentionOptions().SetBufferFuture(10 * time.Minute)
		options.RetentionOptions().SetBufferPast(10 * time.Minute)
		options.RetentionOptions().SetBlockDataExpiry(true)
		options.RetentionOptions().SetBlockDataExpiryAfterNotAccessedPeriod(5 * time.Minute)

		options.IndexOptions().SetEnabled(true)
		options.IndexOptions().SetBlockSize(options.RetentionOptions().BlockSize())
	default:
		return nil
	}

	return &admin.NamespaceAddRequest{
		Name: r.NamespaceName,
		Options: &protonamespace.NamespaceOptions{
			BootstrapEnabled:  options.BootstrapEnabled(),
			FlushEnabled:      options.FlushEnabled(),
			WritesToCommitLog: options.WritesToCommitLog(),
			CleanupEnabled:    options.CleanupEnabled(),
			RepairEnabled:     options.RepairEnabled(),
			SnapshotEnabled:   options.SnapshotEnabled(),
			RetentionOptions: &protonamespace.RetentionOptions{
				RetentionPeriodNanos:                     options.RetentionOptions().RetentionPeriod().Nanoseconds(),
				BlockSizeNanos:                           options.RetentionOptions().BlockSize().Nanoseconds(),
				BufferFutureNanos:                        options.RetentionOptions().BufferFuture().Nanoseconds(),
				BufferPastNanos:                          options.RetentionOptions().BufferPast().Nanoseconds(),
				BlockDataExpiry:                          options.RetentionOptions().BlockDataExpiry(),
				BlockDataExpiryAfterNotAccessPeriodNanos: options.RetentionOptions().BlockDataExpiryAfterNotAccessedPeriod().Nanoseconds(),
			},
			IndexOptions: &protonamespace.IndexOptions{
				Enabled:        options.IndexOptions().Enabled(),
				BlockSizeNanos: options.IndexOptions().BlockSize().Nanoseconds(),
			},
		},
	}
}

func defaultedPlacementInitRequest(r *admin.DatabaseCreateRequest) *admin.PlacementInitRequest {
	var (
		numShards         int32
		replicationFactor int32
		instances         []*placementpb.Instance
	)

	switch r.Type {
	case dbTypeLocal:
		numShards = 16
		replicationFactor = 1
		instances = []*placementpb.Instance{
			&placementpb.Instance{
				Id:             "localhost",
				IsolationGroup: "local",
				Zone:           "embedded",
				Weight:         1,
				Endpoint:       "http://localhost:9000",
				Hostname:       "localhost",
				Port:           9000,
			},
		}
	default:
		// This function assumes the type is valid
		return nil
	}

	return &admin.PlacementInitRequest{
		NumShards:         numShards,
		ReplicationFactor: replicationFactor,
		Instances:         instances,
	}
}
