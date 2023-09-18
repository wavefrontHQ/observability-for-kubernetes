.PHONY=setup lint test start troll-server-container start-troll-server-container
.PHONY=start-resource-verticle start-gateway-host ensure-postgres setup
.PHONY: tmp/dbconfig
LM_VERSION=1.0.11-SNAPSHOT
export JAVA_HOME=/Library/Java/JavaVirtualMachines/temurin-11.jdk/Contents/Home/

troll_server_files=$(shell find troll-server/src)
troll_server_port=8081
cred_files=tmp/csp_username tmp/csp_password
homebrew_etc_dir=$(shell brew --prefix)/etc

tmp:
	mkdir -p tmp

setup:
	./scripts/setup/brew.sh
	./scripts/setup/concourse.sh
	./scripts/setup/check-direnv.sh
	./scripts/setup/kubectl.sh
	./scripts/setup/helm-repos.sh

lint: .vale/styles/write-good
	./scripts/test/lint.sh
	./scripts/test/concourse-lint.sh
	vale --minAlertLevel=warning README.md

.vale/styles/write-good:
	vale sync

clean:
	cd le-mans; mvn clean

start-echo-server:
	python3 scripts/le-mans/echo-server.py $(troll_server_port)

## Le Mans
ensure-cred-files: | tmp
	touch tmp/csp_username
	touch tmp/csp_password
	echo "Template credentials files available at `pwd`/tmp"

ensure-postgres: tmp/dbconfig
	brew services | grep postgresql@14 | grep started || brew services start postgresql@14

ensure-kafka: stop-kafka
	docker-compose up --detach

stop-kafka:
	docker-compose down

start-resource-verticle: ensure-postgres le-mans/le-mans-resources/target/le-mans-resources-${LM_VERSION}-exec.jar
	java -jar le-mans/le-mans-resources/target/le-mans-resources-${LM_VERSION}-exec.jar \
		--bindAddress=0.0.0.0 --port=8001 --isAuthorizationEnabled=true \
		--sandbox=tmp/sandbox --dbConfigFile=tmp/dbconfig \
		--cspUri=https://console-stg.cloud.vmware.com \
		--enableJwksCspKeyFormat=true

start-gateway-host: $(cred_files) ensure-kafka le-mans/le-mans-gateway/target/le-mans-gateway-${LM_VERSION}-exec.jar
	java -jar le-mans/le-mans-gateway/target/le-mans-gateway-${LM_VERSION}-exec.jar \
		--bindAddress=0.0.0.0 --port=8002 --resourcesHostUri=http://localhost:8001 \
		--cspClientId="$(shell cat tmp/csp_username)" --cspClientSecret='$(shell cat tmp/csp_password)' \
		--cspUri=https://console-stg.cloud.vmware.com --enableJwksCspKeyFormat=true

le-mans/le-mans-resources/target/le-mans-resources-${LM_VERSION}-exec.jar:
	cd le-mans; mvn -s tools/build/maven-settings.xml clean install -DskipTests

le-mans/le-mans-gateway/target/le-mans-gateway-${LM_VERSION}-exec.jar:
	cd le-mans; mvn -s tools/build/maven-settings.xml clean install -DskipTests

tmp/dbconfig: | tmp
	./scripts/create-db-config.sh > $@

tmp/csp_username: | tmp
	op read 'op://AOA-TAS2TO/Lemans CSP Credentials/username' -o $@

tmp/csp_password: | tmp
	op read 'op://AOA-TAS2TO/Lemans CSP Credentials/password' -o $@
