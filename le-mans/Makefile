.PHONY=dockerize-lemans start-lemans clean stop-lemans tmp/dbconfig
LM_VERSION=1.0.11-SNAPSHOT
export JAVA_HOME=/Library/Java/JavaVirtualMachines/temurin-11.jdk/Contents/Home/

tmp:
	mkdir -p tmp

start-lemans: tmp/csp_username tmp/csp_password dockerize-lemans stop-lemans tmp/dbconfig
	@CSP_CLIENT_ID='$(shell cat tmp/csp_username)' CSP_CLIENT_SECRET='$(shell cat tmp/csp_password)' docker compose up --abort-on-container-exit

stop-lemans:
	@CSP_CLIENT_ID="" CSP_CLIENT_SECRET="" docker compose down

dockerize-lemans: le-mans/le-mans-resources/target/le-mans-resources-${LM_VERSION}-exec.jar le-mans/le-mans-gateway/target/le-mans-gateway-${LM_VERSION}-exec.jar le-mans/docker/resources/base/lib le-mans/docker/gateway/base/lib
	cp le-mans/le-mans-resources/target/le-mans-resources-${LM_VERSION}-exec.jar le-mans/docker/resources/base/lib/le-mans-resources.jar
	docker build -t lemans/lemans-resources le-mans/docker/resources
	cp le-mans/le-mans-gateway/target/le-mans-gateway-${LM_VERSION}-exec.jar le-mans/docker/gateway/base/lib/le-mans-gateway.jar
	docker build -t lemans/lemans-gateway le-mans/docker/gateway

start-wf-proxy: dockerize-wf-proxy
	docker run --publish 2878:2878  --expose 2878 --env WAVEFRONT_URL="https://nimba.wavefront.com/api" --env WAVEFRONT_TOKEN="c3bbb634-cd2f-4e99-b51a-65e51b7f4132" --env WAVEFRONT_PROXY_ARGS="--le-mans-host http://gateway:8002 --le-mans-stream-name helios-test --le-mans-token z76OxPUXKtwdp48HXLAyvFta6eEov9Eo" --network le-mans_default wf-proxy

dockerize-wf-proxy: wavefront-proxy
	docker build -t wf-proxy -f wavefront-proxy/docker/Dockerfile ./wavefront-proxy

wavefront-proxy:
	git clone git@gitlab.eng.vmware.com:cpo-team-troll/wavefront-proxy.git

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

tmp/csp_username: | tmp
	@echo "Please create a tmp/csp_username file populated with a valid CSP oauth app id"
	@exit 1

tmp/csp_password: | tmp
	@echo "Please create a tmp/csp_password file populated with a valid CSP oauth app secret"
	@exit 1

le-mans:
	git clone https://$(shell whoami)@bellevue-ci-gerrit.eng.vmware.com/le-mans

create-buffered-stream:
	./scripts/create-lemans-buffered-pipelines.sh

post-to-stream:
	./scripts/post-lemans-metric.sh
