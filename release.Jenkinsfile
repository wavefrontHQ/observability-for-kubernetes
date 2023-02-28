pipeline {
  agent any

  tools {
    go 'Go 1.18'
  }

  environment {
    GIT_BRANCH = "main"
    GIT_CREDENTIAL_ID = 'wf-jenkins-github'
    GITHUB_TOKEN = credentials('GITHUB_TOKEN')
    WAVEFRONT_TOKEN = credentials("WAVEFRONT_TOKEN_NIMBA")
    GKE_CLUSTER_NAME = "k8po-jenkins-ci-2"
    GCP_ZONE="a"
    GCP_CREDS = credentials("GCP_CREDS")
    GCP_PROJECT = "wavefront-gcp-dev"
    HARBOR_CREDS = credentials("projects-registry-vmware-tanzu_observability-robot")
    PREFIX = "projects.registry.vmware.com"
  }

  stages {
    stage("Promote release images and test") {
      options {
        timeout(time: 30, unit: 'MINUTES')
      }
      steps {
        withEnv(["PATH+EXTRA=${HOME}/go/bin", "PATH+GCLOUD=${HOME}/google-cloud-sdk/bin"]) {
          sh 'cd operator && ./hack/jenkins/setup-for-integration-test.sh'
          sh 'cd operator && echo $HARBOR_CREDS_PSW | docker login $PREFIX -u $HARBOR_CREDS_USR --password-stdin'
          sh './scripts/promote-release-images.sh'
          lock("integration-test-gke-2") {
            sh 'cd operator && make gke-connect-to-cluster'
            sh 'cd operator && make clean-cluster'
            sh 'cd operator && ./hack/test/deploy/deploy-local.sh -t $WAVEFRONT_TOKEN'
            sh 'cd operator && ./hack/test/run-e2e-tests.sh -t $WAVEFRONT_TOKEN -r advanced -v $(cat release/OPERATOR_VERSION)'
            sh 'cd operator && make clean-cluster'
          }
          sh 'git config --global user.email "svc.wf-jenkins@vmware.com"'
          sh 'git config --global user.name "svc.wf-jenkins"'
          sh 'git remote set-url origin https://${GITHUB_TOKEN}@github.com/wavefronthq/observability-for-kubernetes.git'
          sh 'cd operator && ./hack/jenkins/merge-version-bump.sh'
//           TODO: Uncomment release notes
//           sh 'cd operator && ./hack/jenkins/generate-github-release.sh'
        }
      }
    }
  }

//           TODO: Uncomment post to slack
//   post {
//     regression {
//       slackSend (channel: '#tobs-k8po-team', color: '#FF0000', message: "RELEASE BUILD FAILED: <${env.BUILD_URL}|${env.JOB_NAME} [${env.BUILD_NUMBER}]>")
//     }
//     success {
//         script {
//           BUILD_VERSION = readFile('./operator/release/OPERATOR_VERSION').trim()
//           slackSend (channel: '#tobs-k8po-team', color: '#008000', message: "Success!! `observability-for-kubernetes:v${BUILD_VERSION}` is ready to be released!")
//         }
//     }
//   }
}
