export WEAVIATE_ADDR="http://192.168.219.153:9035"
# export OLLAMA_ADDR="http://192.168.219.140:11434" # homin-vraptor
export OLLAMA_ADDR="http://192.168.219.146:11434" # homin-pc

go build
./processor -ws wss://homin.dev/webchat-relay/ws $@
