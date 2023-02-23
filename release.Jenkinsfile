pipeline {
  agent any

  tools {
    go 'Go 1.17'
  }

  environment {
    OPERATOR_BUMP_COMPONENT = "${params.OPERATOR_BUMP_COMPONENT}"
    COLLECTOR_BUMP_COMPONENT = "${params.COLLECTOR_BUMP_COMPONENT}"
    GIT_BRANCH = "main"
    GIT_CREDENTIAL_ID = 'wf-jenkins-github'
    GITHUB_TOKEN = credentials('GITHUB_TOKEN')
  }

  stages {
    stage("Promote release images") {
      steps {
        withEnv(["PATH+GCLOUD=${HOME}/google-cloud-sdk/bin"]) {
          sh 'cd operator && ./hack/jenkins/setup-for-integration-test.sh'
          sh './scripts/promote-release-images.sh'
        }
      }
    }
    stage("Deploy and test release images") {
      environment {
        WAVEFRONT_TOKEN = credentials("WAVEFRONT_TOKEN_NIMBA")
        GKE_CLUSTER_NAME = "k8po-jenkins-ci-zone-a"
        GCP_ZONE="a"
        GCP_CREDS = credentials("GCP_CREDS")
        GCP_PROJECT = "wavefront-gcp-dev"
        INTEGRATION_TEST_ARGS="-r advanced"
      }
      steps {
        lock("integration-test-gke") {
          sh 'cd operator && make gke-connect-to-cluster'
          sh 'cd operator && make clean-cluster'
          sh 'cd operator && ./hack/test/deploy/deploy-local.sh -t $WAVEFRONT_TOKEN'
          sh 'cd operator && make integration-test'
          sh 'cd operator && make clean-cluster'
        }
      }
    }
//     stage("Merge release version changes and create PR") {
//       steps {
//         sh 'cd operator && ./hack/jenkins/merge-version-bump.sh'
//         sh 'cd operator && ./hack/jenkins/generate-github-release.sh'
//       }
//     }
  }

  post {
    regression {
      slackSend (channel: '#tobs-k8po-team', color: '#FF0000', message: "#TESTING# RELEASE BUILD FAILED: <${env.BUILD_URL}|${env.JOB_NAME} [${env.BUILD_NUMBER}]>")
    }
    success {
        script {
          BUILD_VERSION = readFile('./operator/release/OPERATOR_VERSION').trim()
          slackSend (channel: '#tobs-k8po-team', color: '#008000', message: "#TESTING# Success!! `observability-for-kubernetes:v${BUILD_VERSION}` is ready to be released!")
        }
    }
  }
}
