package client

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/lixichongAAA/seckill/pb"
	"github.com/opentracing/opentracing-go"
	zipkin "github.com/openzipkin-contrib/zipkin-go-opentracing"
)

func TestUserClientImpl_CheckUser(t *testing.T) {
	client, _ := NewUserClient("user", nil, genTracerAct(nil))

	if response, err := client.CheckUser(context.Background(), nil, &pb.UserRequest{
		Username: "xuan",
		Password: "xuan",
	}); err == nil {
		fmt.Println(response.Result)
	} else {
		fmt.Println(err.Error())
	}
}

func genTracerAct(tracer opentracing.Tracer) opentracing.Tracer {
	if tracer != nil {
		return tracer
	}
	zipkinUrl := "http://127.0.0.1:9411/api/v2/spans"
	zipkinRecorder := "localhost:12344"
	collector, err := zipkin.NewHTTPCollector(zipkinUrl)
	if err != nil {
		log.Fatalf("zipkin.NewHTTPCollector err: %v", err)
	}

	recorder := zipkin.NewRecorder(collector, false, zipkinRecorder, "user-client")

	res, err := zipkin.NewTracer(
		recorder, zipkin.ClientServerSameSpan(true),
	)
	if err != nil {
		log.Fatalf("zipkin.NewTracer err: %v", err)
	}
	return res

}
