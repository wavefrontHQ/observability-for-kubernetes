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
    WAVEFRONT_TOKEN = credentials("WAVEFRONT_TOKEN_NIMBA")
    GKE_CLUSTER_NAME = "k8po-jenkins-ci-zone-a"
    GCP_ZONE="a"
    GCP_CREDS = credentials("GCP_CREDS")
    GCP_PROJECT = "wavefront-gcp-dev"
    INTEGRATION_TEST_ARGS="-r advanced"
//     HARBOR_CREDS = credentials("projects-registry-vmware-tanzu_observability_keights_saas-robot")
//     PREFIX = "projects.registry.vmware.com"
  }

  stages {
    stage("Promote release images") {
      steps {
        withEnv(["PATH+EXTRA=${HOME}/go/bin", "PATH+GCLOUD=${HOME}/google-cloud-sdk/bin"]) {
          sh(script: 'cd operator && ./hack/jenkins/setup-for-integration-test.sh')
//           sh 'cd operator && ./hack/jenkins/setup-for-integration-test.sh'
          sh 'cd operator && make semver-cli'
//           sh 'cd operator && echo $HARBOR_CREDS_PSW | docker login $PREFIX -u $HARBOR_CREDS_USR --password-stdin'
          sh './scripts/promote-release-images.sh -o ${OPERATOR_BUMP_COMPONENT} -c ${COLLECTOR_BUMP_COMPONENT}'
          lock("integration-test-gke") {
            sh 'cd operator && make gke-connect-to-cluster'
            sh 'cd operator && make clean-cluster'
            sh 'cd operator && ./hack/test/deploy/deploy-local.sh -t $WAVEFRONT_TOKEN'
            sh 'cd operator && make integration-test'
            sh 'cd operator && make clean-cluster'
          }
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
      slackSend (channel: '#cpo-team-helios', color: '#FF0000', message: "#TESTING# RELEASE BUILD FAILED: <${env.BUILD_URL}|${env.JOB_NAME} [${env.BUILD_NUMBER}]>")
    }
    success {
        script {
          BUILD_VERSION = readFile('./operator/release/OPERATOR_VERSION').trim()
          slackSend (channel: '#cpo-team-helios', color: '#008000', message: "#TESTING# Success!! `observability-for-kubernetes:v${BUILD_VERSION}` is ready to be released!")
        }
    }
  }
}
