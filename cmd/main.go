package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"google.golang.org/grpc"

	pb "github.com/jensg-st/keda-test/pkg/externalscaler"
)

type scaler struct {
	counter int
}

var hw scaler
var hw1 scaler

type ExternalScaler struct{}

var push chan string

func pushHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)

	if vars["count"] == "0" {
		hw.counter = 1
	} else {
		hw1.counter = 1
	}

	push <- vars["count"]
	return

}

func upHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)

	if vars["count"] == "0" {
		hw.counter++
		w.Write([]byte(fmt.Sprintf("COUNTER FOR 0: %d", hw.counter)))
	} else {
		hw1.counter++
		w.Write([]byte(fmt.Sprintf("COUNTER FOR 1: %d", hw1.counter)))
	}

	return

}

func downHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)

	if vars["count"] == "0" {
		hw.counter--
		w.Write([]byte(fmt.Sprintf("COUNTER FOR 0: %d", hw.counter)))
	} else {
		hw1.counter--
		w.Write([]byte(fmt.Sprintf("COUNTER FOR 1: %d", hw1.counter)))
	}

	return

}

func rootHandler(w http.ResponseWriter, r *http.Request) {

	g := fmt.Sprintf(">> %s", r.URL.Path)
	w.Write([]byte(g))
	return

}

func main() {

	push = make(chan string)

	fmt.Println("listenting web on :8000")

	r := mux.NewRouter()
	r.HandleFunc("/up/{count}", upHandler)

	r.HandleFunc("/down/{count}", downHandler)

	r.HandleFunc("/push/{count}", pushHandler)

	r.PathPrefix("/").HandlerFunc(rootHandler)

	srv := &http.Server{
		Handler: r,
		Addr:    "0.0.0.0:8000",
	}

	go runGRPC()

	log.Fatal(srv.ListenAndServe())
}

func runGRPC() {

	grpcServer := grpc.NewServer()
	lis, _ := net.Listen("tcp", ":6000")
	pb.RegisterExternalScalerServer(grpcServer, &ExternalScaler{})

	fmt.Println("listenting grpc on :6000")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal(err)
	}

}

func (e *ExternalScaler) IsActive(ctx context.Context, scaledObject *pb.ScaledObjectRef) (*pb.IsActiveResponse, error) {

	service := scaledObject.ScalerMetadata["service"]
	log.Printf("CHECKING ISACTIVE %v\n", service)

	active := false
	if service == "helloworld" {
		if hw.counter > 0 {
			active = true
		}
	} else {
		if hw1.counter > 0 {
			active = true
		}
	}

	fmt.Printf("SET ISACTIVE for %v: %v\n", service, active)

	return &pb.IsActiveResponse{
		Result: active,
	}, nil

}

func (e *ExternalScaler) GetMetricSpec(ctx context.Context, scaledObject *pb.ScaledObjectRef) (*pb.GetMetricSpecResponse, error) {

	service := scaledObject.ScalerMetadata["service"]
	log.Printf("CHECKING GETMETRICPSEC %v\n", service)

	return &pb.GetMetricSpecResponse{
		MetricSpecs: []*pb.MetricSpec{{
			MetricName: "metric",
			TargetSize: 1,
		}},
	}, nil
}

func (e *ExternalScaler) GetMetrics(_ context.Context, metricRequest *pb.GetMetricsRequest) (*pb.GetMetricsResponse, error) {
	service := metricRequest.ScaledObjectRef.ScalerMetadata["service"]

	log.Printf("CHECKING GetMetrics %v\n", service)

	counter := hw1.counter
	if service == "0" {
		counter = hw.counter
	}

	fmt.Printf("SET COUNTER for %v: %v\n", service, counter)

	return &pb.GetMetricsResponse{
		MetricValues: []*pb.MetricValue{{
			MetricName:  "metric",
			MetricValue: int64(counter),
		}},
	}, nil
}

func (e *ExternalScaler) StreamIsActive(scaledObject *pb.ScaledObjectRef, epsServer pb.ExternalScaler_StreamIsActiveServer) error {
	var err error

	service := scaledObject.ScalerMetadata["service"]
	fmt.Printf("START STREAM FOR %v\n", service)

	for {
		select {
		case in := <-push:
			log.Printf("GOT PUSH!!!!! %v = %v\n", service, in)
			if service == "helloworld1" && in == "1" {
				err = epsServer.Send(&pb.IsActiveResponse{
					Result: true,
				})
				fmt.Printf("ERR %v\n", err)
			} else if service == "helloworld" && in == "0" {
				err = epsServer.Send(&pb.IsActiveResponse{
					Result: true,
				})
				fmt.Printf("ERR %v\n", err)
			}

		case <-epsServer.Context().Done():
			// call cancelled
			return nil
		case <-time.Tick(time.Hour * 1):
			if err != nil {
				// log error
			} else {
				err = epsServer.Send(&pb.IsActiveResponse{
					Result: true,
				})
			}
		}
	}
}
