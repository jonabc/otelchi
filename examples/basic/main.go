package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/riandyrn/otelchi"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"

	stdout "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

var tracer oteltrace.Tracer

func main() {
	// initialize trace provider
	tp := initTracerProvider()
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()
	// set global tracer provider & text propagators
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	// initialize tracer
	tracer = otel.Tracer("mux-server")

	// define router
	r := chi.NewRouter()
	r.Use(otelchi.Middleware("my-server", otelchi.WithChiRoutes(r)))
	r.HandleFunc("/users/{id:[0-9]+}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		name := getUser(r.Context(), id)
		reply := fmt.Sprintf("user %s (id %s)\n", name, id)
		w.Write(([]byte)(reply))
	}))

	// serve router
	_ = http.ListenAndServe(":8080", r)
}

func initTracerProvider() *sdktrace.TracerProvider {
	exporter, err := stdout.New(stdout.WithPrettyPrint())
	if err != nil {
		log.Fatal(err)
	}
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceNameKey.String("mux-server"),
		),
	)
	if err != nil {
		log.Fatalf("unable to initialize resource due: %v", err)
	}
	return sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
}

func getUser(ctx context.Context, id string) string {
	_, span := tracer.Start(ctx, "getUser", oteltrace.WithAttributes(attribute.String("id", id)))
	defer span.End()
	if id == "123" {
		return "otelchi tester"
	}
	return "unknown"
}
