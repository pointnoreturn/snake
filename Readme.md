# snake

Telemetry graphs / stats generator for meshtastic node.

Meshtastic Node -> (TCP/USB) -> snake -> Victoria Metrics -> Grafana

Collects useful stats for better performance report over time.


# Development


Recompile protobufs (meshtastic subdmoule)

     rm -rf github.com/meshtastic/go/generated
     protoc -I protobufs --go_out=. protobufs/nanopb.proto $(find protobufs/meshtastic -name '*.proto')