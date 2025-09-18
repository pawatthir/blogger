package grpcclient

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func UnaryClientLoggingInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, resp interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		startTime := time.Now()

		sentMd, _ := metadata.FromOutgoingContext(ctx)

		logGRPCClientRequest(ctx, method, sentMd, req)

		var receivedMd metadata.MD
		opts = append(opts, grpc.Header(&receivedMd))
		err := invoker(ctx, method, req, resp, cc, opts...)
		var statusCode codes.Code
		var statusError any

		if err != nil {
			statusError = err
			statusCode = status.Code(err)
		}

		logGRPCClientResponse(ctx, method, receivedMd, startTime, resp, statusCode, statusError)

		return err
	}
}

func logGRPCClientRequest(ctx context.Context, method string, md metadata.MD, req any) {
	var reqMap map[string]interface{}
	var reqMapErr error

	reqProto, ok := req.(proto.Message)
	if ok {
		reqMap, reqMapErr = protoMessageToMap(reqProto)
		if reqMapErr != nil {
			slog.WarnContext(ctx, "failed to convert request to map", "error", reqMapErr)
		}
	}

	fields := []any{
		slog.String("type", "grpcclient"),
		slog.String("method", method),
		slog.Any("metadata", md),
		slog.Any("body", reqMap),
	}

	slog.InfoContext(ctx, fmt.Sprintf("Sent gRPC Request to %s", method), fields...)
}

func logGRPCClientResponse(ctx context.Context, method string, md metadata.MD, startTime time.Time, resp interface{}, statusCode codes.Code, statusError any) {
	var respMap map[string]interface{}
	var respMapErr error

	respProto, ok := resp.(proto.Message)
	if ok {
		respMap, respMapErr = protoMessageToMap(respProto)
		if respMapErr != nil {
			slog.WarnContext(ctx, "failed to convert response to map", "error", respMapErr)
		}
	}

	fields := []any{
		slog.String("type", "grpcclient"),
		slog.String("method", method),
		slog.Any("metadata", md),
		slog.Any("body", respMap),
		slog.Any("status_code", statusCode),
		slog.Any("error", statusError),
		slog.String("duration", time.Since(startTime).String()),
	}

	msg := fmt.Sprintf("Received gRPC Response from %s", method)
	if statusError != nil {
		slog.ErrorContext(ctx, msg, fields...)
	} else {
		slog.InfoContext(ctx, msg, fields...)
	}
}

func protoMessageToMap(message proto.Message) (map[string]interface{}, error) {
	if message == nil || reflect.ValueOf(message).IsNil() {
		return nil, nil
	}

	m := protojson.MarshalOptions{EmitUnpopulated: true}
	jsonBytes, err := m.Marshal(message)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(jsonBytes, &result)
	if err != nil {
		return nil, err
	}

	maskSensitiveDataUsingStructTag(message, result)
	return result, nil
}

func maskSensitiveDataUsingStructTag(message proto.Message, data map[string]interface{}) {
	value := reflect.ValueOf(message)
	if !value.IsValid() || value.IsZero() {
		return
	}

	value = value.Elem()
	typeOf := value.Type()

	for i := 0; i < value.NumField(); i++ {
		field := typeOf.Field(i)

		if sensitiveTag, ok := field.Tag.Lookup("sensitive"); ok && sensitiveTag == "true" {
			jsonTag := field.Tag.Get("json")
			jsonFieldName := jsonTag
			if commaIdx := strings.Index(jsonTag, ","); commaIdx > -1 {
				jsonFieldName = jsonTag[:commaIdx]
			}

			if value, exists := data[jsonFieldName]; exists {
				strValue, ok := value.(string)
				if ok {
					if len(strValue) >= 2 {
						data[jsonFieldName] = strValue[0:1] + "*****" + strValue[len(strValue)-1:]
					} else if len(strValue) == 1 {
						data[jsonFieldName] = strValue + "*****"
					}
				}
			}
		}
	}
}

func GRPCClientInterceptor() grpc.UnaryClientInterceptor {
	return UnaryClientLoggingInterceptor()
}