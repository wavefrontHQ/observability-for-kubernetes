.PHONY=dockerize-lemans start-lemans clean stop-lemans tmp/dbconfig create-wf-users
LM_VERSION=1.0.11-SNAPSHOT
FDB_PORT=4550
export JAVA_HOME=/Library/Java/JavaVirtualMachines/temurin-11.jdk/Contents/Home/

tmp:
	mkdir -p tmp

start-lemans: tmp/le_mans_csp_client_id tmp/le_mans_csp_client_secret dockerize-lemans tmp/dbconfig cleanup-lemans
	@CSP_CLIENT_ID='$(shell cat tmp/le_mans_csp_client_id)' CSP_CLIENT_SECRET='$(shell cat tmp/le_mans_csp_client_secret)' docker compose up --abort-on-container-exit lemans-gateway

#stop-lemans:
#	@CSP_CLIENT_ID="" CSP_CLIENT_SECRET="" FDB_PORT="${FDB_PORT}" docker compose down

dockerize-lemans: le-mans/le-mans-resources/target/le-mans-resources-${LM_VERSION}-exec.jar le-mans/le-mans-gateway/target/le-mans-gateway-${LM_VERSION}-exec.jar le-mans/docker/resources/base/lib le-mans/docker/gateway/base/lib
	cp le-mans/le-mans-resources/target/le-mans-resources-${LM_VERSION}-exec.jar le-mans/docker/resources/base/lib/le-mans-resources.jar
	docker build -t lemans/lemans-resources le-mans/docker/resources
	cp le-mans/le-mans-gateway/target/le-mans-gateway-${LM_VERSION}-exec.jar le-mans/docker/gateway/base/lib/le-mans-gateway.jar
	docker build -t lemans/lemans-gateway le-mans/docker/gateway

start-wf-proxy: tmp/wavefront_csp_client_id tmp/wavefront_csp_client_secret dockerize-wf-proxy tmp/lemans_stream_name
	docker run --tty --interactive --rm --publish 2878:2878 --expose 2878 --env WAVEFRONT_URL="https://nimba.wavefront.com/api" --env CSP_APP_ID="$(shell cat tmp/wavefront_csp_client_id)" --env CSP_APP_SECRET="$(shell cat tmp/wavefront_csp_client_secret)" --env WAVEFRONT_PROXY_ARGS="--le-mans-host http://gateway:8002 --le-mans-stream-name $(shell cat tmp/lemans_stream_name) --cspBaseUrl https://console-stg.cloud.vmware.com" --network le-mans_default wf-proxy

start-wf-proxy-local: tmp/local_wavefront_token
	docker run --tty --interactive --rm --publish 2878:2878 --expose 2878 --env WAVEFRONT_URL="http://uberserver:8082/api" --env CSP_APP_ID="$(shell cat tmp/wavefront_csp_client_id)" --env CSP_APP_SECRET="$(shell cat tmp/wavefront_csp_client_secret)" --env WAVEFRONT_PROXY_ARGS="--le-mans-host http://gateway:8002 --le-mans-stream-name $(shell cat tmp/lemans_stream_name) --cspBaseUrl https://console-stg.cloud.vmware.com" --network le-mans_default wf-proxy

dockerize-wf-proxy: wavefront-proxy
	docker build -t wf-proxy -f wavefront-proxy/docker/Dockerfile ./wavefront-proxy

wavefront-proxy:
	git clone --branch k8ssaas-2475-push-metrics-to-le-mans git@github.com:wavefrontHQ/wavefront-proxy.git

start-wf-backend: tmp/fdb-data
	FDB_PORT="${FDB_PORT}" docker-compose up uberserver --abort-on-container-exit
	docker exec le-mans-fdb-1 fdbcli --exec 'configure new single memory'

tmp/fdb-data: tmp
	mkdir -p tmp/fdb-data

test-proxy: wavefront-proxy
	mvn -f wavefront-proxy/proxy test

post-to-proxy:
	./scripts/post-proxy-metric.sh

le-mans/docker/resources/base/lib:
	mkdir $@

le-mans/docker/gateway/base/lib:
	mkdir $@

le-mans/le-mans-resources/target/le-mans-resources-${LM_VERSION}-exec.jar: le-mans
	cd le-mans; mvn -s tools/build/maven-settings.xml clean install -DskipTests

le-mans/le-mans-gateway/target/le-mans-gateway-${LM_VERSION}-exec.jar: le-mans
	cd le-mans; mvn -s tools/build/maven-settings.xml clean install -DskipTests

tmp/dbconfig: | tmp
	./scripts/create-db-config.sh > $@

tmp/local_wavefront_token: | tmp
	@echo "Please create a tmp/local_wavefront_token file and populate it with a valid token"
	@exit 1

cleanup-lemans:
	rm -f tmp/lemans_token
	rm -f tmp/lemans_stream_name

tmp/wavefront_csp_client_id: | tmp
	@echo "Please create a tmp/wavefront_csp_client_id file populated with a valid CSP oauth app id with permission to create proxies and permission to push data to le mans"
	@exit 1

tmp/wavefront_csp_client_secret: | tmp
	@echo "Please create a tmp/wavefront_csp_client_secret file populated with a valid CSP oauth app secret"
	@exit 1

tmp/le_mans_csp_client_id: | tmp
	@echo "Please create a tmp/le_mans_csp_client_id file populated with a valid CSP oauth app id that is allow listed with le mans"
	@exit 1

tmp/le_mans_csp_client_secret: | tmp
	@echo "Please create a tmp/le_mans_csp_client_secret file populated with a valid CSP oauth app secret"
	@exit 1

tmp/lemans_stream_name: | tmp
	@echo "Please create a tmp/lemans_stream_name file populated with a valid Lemans stream name"
	@exit 1

tmp/wf_password: | tmp
	@echo "Please create a tmp/wf_password file populated with some password (like Wavefront1!)"
	@exit 1

le-mans:
	git clone https://$(shell whoami)@bellevue-ci-gerrit.eng.vmware.com/le-mans

create-buffered-stream: tmp/lemans_stream_name
	STREAM_NAME=$(shell cat tmp/lemans_stream_name) ./scripts/create-lemans-buffered-pipelines.sh

post-to-stream:
	./scripts/post-lemans-metric.sh

setup-wf-customers: system/cli/target/wfcli.sh
	docker cp ./system/cli/target le-mans-uberserver-1:/wf-cli
	docker cp ./tmp/wf_password le-mans-uberserver-1:/tmp/wf_password
	docker cp ./scripts le-mans-uberserver-1:/scripts
	docker exec le-mans-uberserver-1 /scripts/setup-wf-customers.sh

system/cli/target/wfcli.sh: system
	cd system && mvn install -DskipTests=true -Duse.github.diff=none

system:
	git clone --branch k8ssaas-2427-ingest-from-le-mans git@github.com:sunnylabs/system.git
