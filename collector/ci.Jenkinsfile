pipeline {
  agent {
    label 'nimbus-cloud'
  }
  environment {
    GITHUB_CREDS_PSW = credentials("GITHUB_TOKEN")
  }

  stages {
    stage("Test with Go 1.18") {
      tools {
        go 'Go 1.18'
      }
      steps {
        withEnv(["PATH+EXTRA=${HOME}/go/bin"]) {
            sh 'cd collector && make checkfmt vet tests'
        }
      }
    }

    stage("Build and publish") {
      parallel{
        stage("Test Openshift build") {
          steps {
            sh 'cd collector && docker build -f deploy/docker/Dockerfile-rhel .'
          }
        }
        stage("Publish") {
          tools {
            go 'Go 1.18'
          }
          environment {
            RELEASE_TYPE = "alpha"
            VERSION_POSTFIX = "-alpha-${GIT_COMMIT.substring(0, 8)}"
            HARBOR_CREDS = credentials("projects-registry-vmware-tanzu_observability_keights_saas-robot")
            PREFIX = "projects.registry.vmware.com/tanzu_observability_keights_saas"
            DOCKER_IMAGE = "kubernetes-collector-snapshot"
          }
          steps {
            withEnv(["PATH+EXTRA=${HOME}/go/bin"]) {
               sh 'cd collector && ./hack/jenkins/install_docker_buildx.sh'
               sh 'cd collector && make semver-cli'
               sh 'echo $HARBOR_CREDS_PSW | docker login $PREFIX -u $HARBOR_CREDS_USR --password-stdin'
               sh 'cd collector && HARBOR_CREDS_USR=$(echo $HARBOR_CREDS_USR | sed \'s/\\$/\\$\\$/\') make publish'
            }
          }
        }
      }
    }

    stage('Run Integration Tests') {
      // To save time, the integration tests and wavefront-metrics tests are split up between gke and eks
      // But we want to make sure that the combined and default integration tests are run on both
      parallel {
        stage("GKE Integration Test") {
          agent {
            label "gke"
          }
          options {
            timeout(time: 20, unit: 'MINUTES')
          }
          tools {
            go 'Go 1.18'
          }
          environment {
            GCP_CREDS = credentials("GCP_CREDS")
            GKE_CLUSTER_NAME = "k8po-jenkins-ci-zone-a"
            GCP_ZONE="a"
            VERSION_POSTFIX = "-alpha-${GIT_COMMIT.substring(0, 8)}"
            PREFIX = "projects.registry.vmware.com/tanzu_observability_keights_saas"
            DOCKER_IMAGE = "kubernetes-collector-snapshot"
            WAVEFRONT_TOKEN = credentials("WAVEFRONT_TOKEN_NIMBA")
            INTEGRATION_TEST_ARGS="all"
            INTEGRATION_TEST_BUILD="ci"
          }
          steps {
            withEnv(["PATH+GO=${HOME}/go/bin", "PATH+GCLOUD=${HOME}/google-cloud-sdk/bin"]) {
              lock("integration-test-gke") {
                sh 'cd collector && ./hack/jenkins/setup-for-integration-test.sh -k gke'
                sh 'cd collector && make gke-connect-to-cluster'
                sh 'cd collector && make clean-cluster'
                sh 'cd collector && make integration-test'
                sh 'cd collector && make clean-cluster'
              }
            }
          }
        }
        stage("EKS Integration Test") {
          agent {
            label "eks"
          }
          options {
            timeout(time: 20, unit: 'MINUTES')
          }
          tools {
            go 'Go 1.18'
          }
          environment {
            VERSION_POSTFIX = "-alpha-${GIT_COMMIT.substring(0, 8)}"
            PREFIX = "projects.registry.vmware.com/tanzu_observability_keights_saas"
            DOCKER_IMAGE = "kubernetes-collector-snapshot"
            AWS_SHARED_CREDENTIALS_FILE = credentials("k8po-ci-aws-creds")
            AWS_CONFIG_FILE = credentials("k8po-ci-aws-profile")
            WAVEFRONT_TOKEN = credentials("WAVEFRONT_TOKEN_NIMBA")
            INTEGRATION_TEST_ARGS="all"
            INTEGRATION_TEST_BUILD="ci"
          }
          steps {
            withEnv(["PATH+GO=${HOME}/go/bin"]) {
              lock("integration-test-eks") {
                sh 'cd collector && ./hack/jenkins/setup-for-integration-test.sh -k eks'
                sh 'cd collector && make target-eks'
                sh 'cd collector && make clean-cluster'
                sh 'cd collector && make integration-test'
                sh 'cd collector && make clean-cluster'
              }
            }
          }
        }
        stage("AKS Integration Test") {
          agent {
            label "aks"
          }
          options {
            timeout(time: 30, unit: 'MINUTES')
          }
          tools {
            go 'Go 1.18'
          }
          environment {
            AKS_CLUSTER_NAME = "k8po-ci"
            VERSION_POSTFIX = "-alpha-${GIT_COMMIT.substring(0, 8)}"
            PREFIX = "projects.registry.vmware.com/tanzu_observability_keights_saas"
            DOCKER_IMAGE = "kubernetes-collector-snapshot"
            WAVEFRONT_TOKEN = credentials("WAVEFRONT_TOKEN_NIMBA")
            INTEGRATION_TEST_ARGS="real-proxy-metrics"
            INTEGRATION_TEST_BUILD="ci"
          }
          steps {
            withEnv(["PATH+GO=${HOME}/go/bin"]) {
             lock("integration-test-aks") {
               withCredentials([file(credentialsId: 'aks-kube-config', variable: 'KUBECONFIG')]) {
                 sh 'cd collector && ./hack/jenkins/setup-for-integration-test.sh -k aks'
                 sh 'cd collector && kubectl config use k8po-ci'
                 sh 'cd collector && make clean-cluster'
                 sh 'cd collector && make integration-test'
                 sh 'cd collector && make clean-cluster'
               }
             }
            }
          }
        }
      }
    }
  }

  post {
    regression {
      slackSend (channel: '#tobs-k8po-team', color: '#FF0000', message: "CI BUILD FAILED: <${env.BUILD_URL}|${env.JOB_NAME} [${env.BUILD_NUMBER}]>")
    }
    fixed {
      slackSend (channel: '#tobs-k8po-team', color: '#008000', message: "CI BUILD FIXED: <${env.BUILD_URL}|${env.JOB_NAME} [${env.BUILD_NUMBER}]>")
    }
  }
}