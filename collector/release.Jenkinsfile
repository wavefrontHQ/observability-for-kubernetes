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
      agent {
        label "worker-1"
      }
      steps {
        withEnv(["PATH+EXTRA=${HOME}/go/bin"]) {
          sh 'cd collector && ./hack/jenkins/install_docker_buildx.sh'
          sh 'cd collector && make semver-cli'
        }
      }
    }
    stage("Create Bump Version Branch") {
      agent {
        label "worker-1"
      }
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
      agent {
        label "worker-1"
      }
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
      agent {
        label "worker-1"
      }
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
      agent {
        label "worker-1"
      }
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
      agent {
        label "worker-1"
      }
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
//     stage("Push Openshift Image to RedHat Connect") {
//       environment {
//         REDHAT_CREDS=credentials('redhat-connect-wf-collector-creds')
//         REDHAT_OSPID=credentials("redhat-connect-ospid-wf-collector")
//         REDHAT_API_KEY=credentials("redhat-connect-api-key")
//         REDHAT_PROJECT_ID=credentials("redhat-connect-collector-project-id")
//         OPENSHIFT_CREDS_PSW=credentials('OPENSHIFT_CREDS_PSW')
//         OPENSHIFT_VM=credentials('OPENSHIFT_VM')
//         GIT_BUMP_BRANCH_NAME = "${sh(script:'git name-rev --name-only HEAD', returnStdout: true).trim()}"
//       }
//       steps {
//           script {
//             env.PREFIX = "quay.io/redhat-isv-containers/${env.REDHAT_PROJECT_ID}"
//           }
//           sh """
//           sshpass -p "${OPENSHIFT_CREDS_PSW}" ssh -o StrictHostKeyChecking=no root@${OPENSHIFT_VM} "bash -s" < hack/jenkins/release-openshift-container.sh \
//                                                                                                                        ${PREFIX} \
//                                                                                                                        ${REDHAT_CREDS_USR} \
//                                                                                                                        ${REDHAT_CREDS_PSW} \
//                                                                                                                        ${REDHAT_API_KEY} \
//                                                                                                                        ${REDHAT_PROJECT_ID} \
//                                                                                                                        ${GIT_BUMP_BRANCH_NAME} \
//                                                                                                                        ${RC_NUMBER}
//           """
//         }
//     }
    stage("Create and Merge Bump Version Pull Request") {
      agent {
        label "worker-1"
      }
      steps {
        sh 'cd collector && ./hack/jenkins/create-and-merge-pull-request.sh'
      }
    }
    stage("Github Release") {
      agent {
        label "worker-1"
      }
      environment {
        GITHUB_CREDS_PSW = credentials("GITHUB_TOKEN")
      }
      steps {
        sh 'cd collector && ./hack/jenkins/generate_github_release.sh'
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