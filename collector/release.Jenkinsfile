pipeline {
  agent any

  tools {
    go 'Go 1.18'
  }

  environment {
    RELEASE_TYPE = 'release'
    RC_NUMBER = "1"
    BUMP_COMPONENT = "${params.BUMP_COMPONENT}"
    GIT_BRANCH = getCurrentBranchName()
    GIT_CREDENTIAL_ID = 'wf-jenkins-github'
    TOKEN = credentials('GITHUB_TOKEN')
  }

  stages {
    stage("Setup tools") {
      steps {
        withEnv(["PATH+EXTRA=${HOME}/go/bin"]) {
          sh 'cd collector && ./hack/jenkins/install_docker_buildx.sh'
          sh 'cd collector && make semver-cli'
        }
      }
    }
    stage("Create Bump Version Branch") {
      steps {
        withEnv(["PATH+EXTRA=${HOME}/go/bin"]){
          sh 'git config --global user.email "svc.wf-jenkins@vmware.com"'
          sh 'git config --global user.name "svc.wf-jenkins"'
          sh 'git remote set-url origin https://${TOKEN}@github.com/wavefronthq/observability-for-kubernetes.git'
          sh 'cd collector && ./hack/jenkins/create-bump-version-branch.sh "${BUMP_COMPONENT}"'
        }
      }
    }
    stage("Publish RC Release") {
      environment {
        HARBOR_CREDS = credentials("projects-registry-vmware-tanzu_observability-robot")
        PREFIX = 'projects.registry.vmware.com/tanzu_observability'
        DOCKER_IMAGE = 'kubernetes-collector'
        RELEASE_TYPE = 'rc'
      }
      steps {
        sh 'echo $HARBOR_CREDS_PSW | docker login $PREFIX -u $HARBOR_CREDS_USR --password-stdin'
        sh 'cd collector && HARBOR_CREDS_USR=$(echo $HARBOR_CREDS_USR | sed \'s/\\$/\\$\\$/\') make publish'
      }
    }
    // deploy to GKE and run manual tests
    // now we have confidence in the validity of our RC release
    stage("Deploy and Test") {
      environment {
        GCP_CREDS = credentials("GCP_CREDS")
        GKE_CLUSTER_NAME = "k8po-jenkins-ci-zone-a"
        GCP_ZONE="a"
        WAVEFRONT_TOKEN = credentials("WAVEFRONT_TOKEN_NIMBA")
        WF_CLUSTER = 'nimba'
        RELEASE_TYPE = 'rc'
      }
      steps {
        script {
          env.VERSION = readFile('./collector/release/VERSION').trim()
          env.CURRENT_VERSION = "${env.VERSION}-rc-${env.RC_NUMBER}"
          env.CONFIG_CLUSTER_NAME = "jenkins-${env.CURRENT_VERSION}-test"
        }
        withCredentials([string(credentialsId: 'nimba-wavefront-token', variable: 'WAVEFRONT_TOKEN')]) {
          withEnv(["PATH+GCLOUD=${HOME}/google-cloud-sdk/bin"]) {
            lock("integration-test-gke") {
              sh 'cd collector && ./hack/jenkins/setup-for-integration-test.sh -k gke'
              sh 'cd collector && make gke-connect-to-cluster'
              sh 'cd collector && make clean-cluster'
              sh 'cd collector && ./hack/test/deploy/deploy-local-linux.sh'
              sh 'cd collector && ./hack/test/test-wavefront-metrics.sh -c ${WF_CLUSTER} -t ${WAVEFRONT_TOKEN} -n ${CONFIG_CLUSTER_NAME} -v ${VERSION}'
              sh 'cd collector && make clean-cluster'
            }
          }
        }
      }
    }
    stage("Publish GA Harbor Image") {
      environment {
        HARBOR_CREDS = credentials("projects-registry-vmware-tanzu_observability-robot")
        RELEASE_TYPE = 'release'
        PREFIX = 'projects.registry.vmware.com/tanzu_observability'
        DOCKER_IMAGE = 'kubernetes-collector'
      }
      steps {
        sh 'echo $HARBOR_CREDS_PSW | docker login $PREFIX -u $HARBOR_CREDS_USR --password-stdin'
        sh 'cd collector && HARBOR_CREDS_USR=$(echo $HARBOR_CREDS_USR | sed \'s/\\$/\\$\\$/\') make publish'
      }
    }
    stage("Publish GA Docker Hub") {
      environment {
        DOCKERHUB_CREDS=credentials('Dockerhub_svcwfjenkins')
        RELEASE_TYPE = 'release'
        PREFIX = 'wavefronthq'
        DOCKER_IMAGE = 'wavefront-kubernetes-collector'
      }
      steps {
        sh 'echo $DOCKERHUB_CREDS_PSW | docker login -u $DOCKERHUB_CREDS_USR --password-stdin'
        sh 'cd collector && make publish'
      }
    }
    stage("Create and Merge Bump Version Pull Request") {
      steps {
        sh 'cd collector && ./hack/jenkins/create-and-merge-pull-request.sh'
      }
    }
  }

  post {
    regression {
      slackSend (channel: '#tobs-k8po-team', color: '#FF0000', message: "RELEASE BUILD FAILED: <${env.BUILD_URL}|${env.JOB_NAME} [${env.BUILD_NUMBER}]>")
    }
    success {
        script {
          BUILD_VERSION = readFile('./collector/release/VERSION').trim()
          slackSend (channel: '#tobs-k8po-team', color: '#008000', message: "Success!! `wavefront-collector-for-kubernetes:v${BUILD_VERSION}` is ready to be released!")
        }
    }
  }
}

def getCurrentBranchName() {
  return env.BRANCH_NAME.split("/")[1]
}