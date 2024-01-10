pipeline {
  agent any

  tools {
    go 'Go 1.20'
  }

  environment {
    PATH = "${env.WORKSPACE}/bin:${env.HOME}/go/bin:${env.HOME}/google-cloud-sdk/bin:${env.PATH}"
    GIT_BRANCH = "main"
    GIT_CREDENTIAL_ID = 'wf-jenkins-github'
    GITHUB_TOKEN = credentials('GITHUB_TOKEN')
    WAVEFRONT_TOKEN = credentials("WAVEFRONT_TOKEN_QA4")
    GKE_CLUSTER_NAME = "k8po-jenkins-ci-2"
    GCP_ZONE = "a"
    GCP_CREDS = credentials("GCP_CREDS")
    GCP_PROJECT = "wavefront-gcp-dev"
    HARBOR_CREDS = credentials("projects-registry-vmware-tanzu_observability-robot")
    PREFIX = "projects.registry.vmware.com"
    OPERATOR_VERSION = sh(script: 'cat operator/release/OPERATOR_VERSION', returnStdout: true).trim()
    K8S_CLUSTER_NAME = sh(script: 'echo $(whoami)-$(date +%y%m%d)-release-test', returnStdout: true).trim()
  }

  stages {
    stage("Promote release images and test") {
      options {
        timeout(time: 40, unit: 'MINUTES')
      }
      steps {
        sh './operator/hack/jenkins/setup-for-integration-test.sh'
        sh 'echo $HARBOR_CREDS_PSW | docker login $PREFIX -u $HARBOR_CREDS_USR --password-stdin'
        sh './scripts/promote-release-images.sh'
        lock("integration-test-gke-2") {
          sh 'make gke-connect-to-cluster'
          sh 'NUMBER_OF_NODES=2 GKE_NODE_POOL=default-pool make resize-node-pool-gke-cluster'
          sh 'NUMBER_OF_NODES=1 GKE_NODE_POOL=arm-pool make resize-node-pool-gke-cluster'
          sh 'CLEAN_CLUSTER_ARGS="-n" make clean-cluster'
          sh './operator/hack/test/deploy/deploy-local.sh -t $WAVEFRONT_TOKEN -n $K8S_CLUSTER_NAME -x'
          sh './operator/hack/test/run-e2e-tests.sh -t $WAVEFRONT_TOKEN -r basic -v $(cat operator/release/OPERATOR_VERSION) -n $K8S_CLUSTER_NAME'
          sh 'make clean-cluster'
          sh 'NUMBER_OF_NODES=0 GKE_NODE_POOL=default-pool make resize-node-pool-gke-cluster GKE_ASYNC=true'
          sh 'NUMBER_OF_NODES=0 GKE_NODE_POOL=arm-pool make resize-node-pool-gke-cluster GKE_ASYNC=true'
        }
        sh 'git config --global user.email "svc.wf-jenkins@vmware.com"'
        sh 'git config --global user.name "svc.wf-jenkins"'
        sh 'git remote set-url origin https://${GITHUB_TOKEN}@github.com/wavefronthq/observability-for-kubernetes.git'
        sh './operator/hack/jenkins/merge-version-bump.sh'
        sh './operator/hack/jenkins/generate-github-release.sh'
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
          slackSend (channel: '#tobs-k8po-team', color: '#008000', message: "Success!! `observability-for-kubernetes:v${BUILD_VERSION}` is released!")
        }
    }
  }
}
