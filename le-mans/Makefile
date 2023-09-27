.PHONY=dockerize-lemans start-lemans clean stop-lemans tmp/dbconfig
LM_VERSION=1.0.11-SNAPSHOT
export JAVA_HOME=/Library/Java/JavaVirtualMachines/temurin-11.jdk/Contents/Home/

tmp:
	mkdir -p tmp

start-lemans: tmp/wavefront_csp_client_id tmp/wavefront_csp_client_secret dockerize-lemans stop-lemans tmp/dbconfig cleanup-lemans
	@CSP_CLIENT_ID='$(shell cat tmp/le_mans_csp_client_id)' CSP_CLIENT_SECRET='$(shell cat tmp/le_mans_csp_client_secret)' docker compose up --abort-on-container-exit

stop-lemans:
	@CSP_CLIENT_ID="" CSP_CLIENT_SECRET="" docker compose down

dockerize-lemans: le-mans/le-mans-resources/target/le-mans-resources-${LM_VERSION}-exec.jar le-mans/le-mans-gateway/target/le-mans-gateway-${LM_VERSION}-exec.jar le-mans/docker/resources/base/lib le-mans/docker/gateway/base/lib
	cp le-mans/le-mans-resources/target/le-mans-resources-${LM_VERSION}-exec.jar le-mans/docker/resources/base/lib/le-mans-resources.jar
	docker build -t lemans/lemans-resources le-mans/docker/resources
	cp le-mans/le-mans-gateway/target/le-mans-gateway-${LM_VERSION}-exec.jar le-mans/docker/gateway/base/lib/le-mans-gateway.jar
	docker build -t lemans/lemans-gateway le-mans/docker/gateway

start-wf-proxy: dockerize-wf-proxy tmp/lemans_stream_name
	docker run --tty --interactive --rm --publish 2878:2878 --expose 2878 --env WAVEFRONT_URL="https://nimba.wavefront.com/api" --env CSP_APP_ID="$(shell cat tmp/wavefront_csp_client_id)" --env CSP_APP_SECRET="$(shell cat tmp/wavefront_csp_client_secret)" --env WAVEFRONT_PROXY_ARGS="--le-mans-host http://gateway:8002 --le-mans-stream-name $(shell cat tmp/lemans_stream_name) --cspBaseUrl https://console-stg.cloud.vmware.com" --network le-mans_default wf-proxy

dockerize-wf-proxy: wavefront-proxy
	docker build -t wf-proxy -f wavefront-proxy/docker/Dockerfile ./wavefront-proxy

wavefront-proxy:
	git clone git@gitlab.eng.vmware.com:cpo-team-troll/wavefront-proxy.git

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

le-mans:
	git clone https://$(shell whoami)@bellevue-ci-gerrit.eng.vmware.com/le-mans

create-buffered-stream:
	./scripts/create-lemans-buffered-pipelines.sh

post-to-stream:
	./scripts/post-lemans-metric.sh
