pipeline {
  agent any

  tools {
    go 'Go 1.19'
  }

  environment {
    BUMP_COMPONENT = "${params.BUMP_COMPONENT}"
    GIT_BRANCH = "main"
    GIT_CREDENTIAL_ID = 'wf-jenkins-github'
    GITHUB_TOKEN = credentials('GITHUB_TOKEN')
  }

  stages {
    stage("Setup tools") {
      steps {
        withEnv(["PATH+EXTRA=${HOME}/go/bin"]) {
          sh 'cd operator && ./hack/jenkins/install_docker_buildx.sh'
          sh 'cd operator && make semver-cli'
        }
      }
    }
    stage("Create Bump Version Branch") {
      steps {
        withEnv(["PATH+EXTRA=${HOME}/go/bin"]){
          sh 'git config --global user.email "svc.wf-jenkins@vmware.com"'
          sh 'git config --global user.name "svc.wf-jenkins"'
          sh 'git remote set-url origin https://${GITHUB_TOKEN}@github.com/wavefronthq/observability-for-kubernetes.git'
          sh 'cd operator && ./hack/jenkins/bump-version.sh -s "${BUMP_COMPONENT}"'
        }
      }
    }
    stage("Promote Image and Generate YAML") {
      environment {
        HARBOR_CREDS = credentials("projects-registry-vmware-tanzu_observability-robot")
        PREFIX = 'projects.registry.vmware.com/tanzu_observability'
        DOCKER_IMAGE = 'kubernetes-operator'
      }
      steps {
        script {
          env.VERSION = readFile('./operator/release/OPERATOR_VERSION').trim()
        }
        sh 'echo $HARBOR_CREDS_PSW | docker login $PREFIX -u $HARBOR_CREDS_USR --password-stdin'
        sh 'cd operator && make released-kubernetes-yaml'
      }
    }
    // deploy to GKE and run manual tests
    stage("Deploy and Test") {
      environment {
        GCP_CREDS = credentials("GCP_CREDS")
        GKE_CLUSTER_NAME = "k8po-jenkins-ci-zone-a"
        GCP_ZONE="a"
        WAVEFRONT_TOKEN = credentials("WAVEFRONT_TOKEN_NIMBA")
        WF_CLUSTER = 'nimba'
      }
      steps {
        script {
          env.VERSION = readFile('./operator/release/OPERATOR_VERSION').trim()
        }
        withEnv(["PATH+GO=${HOME}/go/bin", "PATH+GCLOUD=${HOME}/google-cloud-sdk/bin"]) {
          lock("integration-test-gke") {
            sh 'cd operator && ./hack/jenkins/setup-for-integration-test.sh'
            sh 'cd operator && make gke-connect-to-cluster'
            sh 'cd operator && make clean-cluster'
            sh 'cd operator && ./hack/test/deploy/deploy-local.sh -t $WAVEFRONT_TOKEN'
            sh 'cd operator && ./hack/test/run-e2e-tests.sh -t $WAVEFRONT_TOKEN -r advanced'
            sh 'cd operator && make clean-cluster'
          }
        }
      }
    }
    stage("Merge bumped versions") {
      steps {
        sh 'cd operator && ./hack/jenkins/merge-version-bump.sh'
      }
    }
    stage("Github Release") {
      steps {
        sh 'cd operator && ./hack/jenkins/generate-github-release.sh'
      }
    }
  }

  post {
    regression {
      slackSend (channel: '#tobs-k8po-team', color: '#FF0000', message: "RELEASE BUILD FAILED: <${env.BUILD_URL}|${env.JOB_NAME} [${env.BUILD_NUMBER}]>")
    }
    success {
        script {
          BUILD_VERSION = readFile('./operator/release/OPERATOR_VERSION').trim()
          slackSend (channel: '#tobs-k8po-team', color: '#008000', message: "Success!! `observability-for-kubernetes:v${BUILD_VERSION}` is ready to be released!")
        }
    }
  }
}
