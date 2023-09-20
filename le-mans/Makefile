.PHONY=dockerize-lemans
.PHONY: tmp/dbconfig
LM_VERSION=1.0.11-SNAPSHOT
export JAVA_HOME=/Library/Java/JavaVirtualMachines/temurin-11.jdk/Contents/Home/

troll_server_files=$(shell find troll-server/src)
troll_server_port=8000
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
	python3 scripts/echo-server.py $(troll_server_port)

## Le Mans
ensure-cred-files: | tmp
	touch tmp/csp_username
	touch tmp/csp_password
	echo "Template credentials files available at `pwd`/tmp"

start-lemans: dockerize-lemans | stop-lemans
	docker compose up

stop-lemans:
	docker compose down

dockerize-lemans: le-mans/le-mans-resources/target/le-mans-resources-${LM_VERSION}-exec.jar le-mans/le-mans-gateway/target/le-mans-gateway-${LM_VERSION}-exec.jar le-mans/docker/resources/base/lib le-mans/docker/gateway/base/lib
	cp le-mans/le-mans-resources/target/le-mans-resources-${LM_VERSION}-exec.jar le-mans/docker/resources/base/lib/le-mans-resources.jar
	docker build -t lemans/lemans-resources le-mans/docker/resources
	cp le-mans/le-mans-gateway/target/le-mans-gateway-${LM_VERSION}-exec.jar le-mans/docker/gateway/base/lib/le-mans-gateway.jar
	docker build -t lemans/lemans-gateway le-mans/docker/gateway

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
	op read 'op://AOA-TAS2TO/Lemans CSP Credentials/username' -o $@

tmp/csp_password: | tmp
	op read 'op://AOA-TAS2TO/Lemans CSP Credentials/password' -o $@

le-mans:
	git clone https://$(shell whoami)@bellevue-ci-gerrit.eng.vmware.com/le-mans