pipeline {
  agent {
    label 'nimbus-cloud'
  }

  tools {
    go 'Go 1.17'
  }

  environment {
    PATH = "${env.HOME}/go/bin:${env.HOME}/google-cloud-sdk/bin:${env.PATH}"
    GITHUB_CREDS_PSW = credentials("GITHUB_TOKEN")
    HARBOR_CREDS = credentials("projects-registry-vmware-tanzu_observability-robot")
    PREFIX = 'projects.registry.vmware.com/tanzu_observability'
    DOCKER_IMAGE = "kubernetes-operator-snapshot"
    VERSION_POSTFIX = "-alpha-${GIT_COMMIT.substring(0, 8)}"
    WAVEFRONT_TOKEN = credentials("WAVEFRONT_TOKEN_NIMBA")
  }

  parameters {
      string(name: 'OPERATOR_YAML_RC_SHA', defaultValue: '')
  }

  stages {
    stage("Run Go tests") {
      parallel{
        stage("Collector Go Tests") {
          tools {
            go 'Go 1.18'
          }
          steps {
            withEnv(["PATH+EXTRA=${HOME}/go/bin"]) {
              sh 'cd collector && make checkfmt vet tests'
            }
          }
        }

        stage("Operator Go Tests") {
          steps {
            sh 'cd operator && make checkfmt vet test'
            sh 'cd operator && make linux-golangci-lint'
            sh 'cd operator && make golangci-lint'
          }
        }
      }
    }

    stage("Build and publish Collector") {
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

    stage('Run Collector Integration Tests') {
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

    stage("Setup For Publishing Operator") {
      environment {
        GCP_CREDS = credentials("GCP_CREDS")
      }
      steps {
        sh 'cd operator && ./hack/jenkins/setup-for-integration-test.sh'
        sh 'cd operator && ./hack/jenkins/install_docker_buildx.sh'
        sh 'cd operator && make semver-cli'
      }
    }

    stage("Publish Operator") {
      environment {
        RELEASE_TYPE = "alpha"
      }
      steps {
        sh 'cd operator && echo $HARBOR_CREDS_PSW | docker login $PREFIX -u $HARBOR_CREDS_USR --password-stdin'
        sh 'cd operator && make docker-xplatform-build'
      }
    }

    stage("Update RC branch") {
      environment {
        RELEASE_TYPE = "alpha"
        TOKEN = credentials('GITHUB_TOKEN')
      }
      steps {
          sh 'operator/hack/jenkins/create-rc-ci.sh'
          script {
            env.OPERATOR_YAML_RC_SHA = sh(returnStdout: true, script: 'git rev-parse HEAD').trim()
          }
      }
    }

    stage("Run Operator Integration Tests") {
      environment {
        OPERATOR_YAML_TYPE="rc"
        TOKEN = credentials('GITHUB_TOKEN')
      }

      parallel {
        stage("GKE") {
          agent {
            label "gke"
          }
          options {
            timeout(time: 30, unit: 'MINUTES')
          }
          environment {
            GKE_CLUSTER_NAME = "k8po-jenkins-ci-zone-a"
            GCP_ZONE="a"
            GCP_CREDS = credentials("GCP_CREDS")
            GCP_PROJECT = "wavefront-gcp-dev"
          }
          stages {
            stage("without customization") {
              steps {
                sh 'cd operator && ./hack/jenkins/setup-for-integration-test.sh'
                sh 'cd operator && ./hack/jenkins/install_docker_buildx.sh'
                sh 'cd operator && make semver-cli'
                lock("integration-test-gke") {
                    sh 'cd operator && make gke-connect-to-cluster'
                    sh 'cd operator && make clean-cluster'
                    sh 'cd operator && make integration-test'
                    sh 'cd operator && make clean-cluster'
                }
              }
            }

            stage("with customization") {
              environment {
                KUSTOMIZATION_TYPE="custom"
                NS="custom-namespace"
                SOURCE_PREFIX="projects.registry.vmware.com/tanzu_observability"
                PREFIX="projects.registry.vmware.com/tanzu_observability_keights_saas"
                HARBOR_CREDS = credentials("projects-registry-vmware-tanzu_observability_keights_saas-robot")
                INTEGRATION_TEST_ARGS="-r advanced"
              }
              steps {
                sh 'cd operator && ./hack/jenkins/setup-for-integration-test.sh'
                sh 'cd operator && ./hack/jenkins/install_docker_buildx.sh'
                sh 'cd operator && make semver-cli'
                lock("integration-test-gke") {
                  sh 'cd operator && make gke-connect-to-cluster'
                  sh 'cd operator && docker logout $PREFIX'
                  sh 'cd operator && echo $HARBOR_CREDS_PSW | docker login $PREFIX -u $HARBOR_CREDS_USR --password-stdin'
                  sh 'cd operator && make docker-copy-images'
                  sh 'cd operator && make integration-test'
                  sh 'cd operator && make clean-cluster'
                }
              }
            }
          }
        }

        stage("EKS") {
          agent {
            label "eks"
          }
          options {
            timeout(time: 30, unit: 'MINUTES')
          }
          environment {
            GCP_CREDS = credentials("GCP_CREDS")
            AWS_SHARED_CREDENTIALS_FILE = credentials("k8po-ci-aws-creds")
            AWS_CONFIG_FILE = credentials("k8po-ci-aws-profile")
          }
          steps {
            sh 'cd operator && ./hack/jenkins/setup-for-integration-test.sh'
            sh 'cd operator && ./hack/jenkins/install_docker_buildx.sh'
            sh 'cd operator && make semver-cli'
            lock("integration-test-eks") {
              sh 'cd operator && make target-eks'
              sh 'cd operator && make clean-cluster'
              sh 'cd operator && make integration-test'
              sh 'cd operator && make clean-cluster'
            }
          }
        }

        stage("AKS") {
          agent {
            label "aks"
          }
          options {
            timeout(time: 30, unit: 'MINUTES')
          }
          environment {
            GCP_CREDS = credentials("GCP_CREDS")
            AKS_CLUSTER_NAME = "k8po-ci"
          }
          steps {
            sh 'cd operator && ./hack/jenkins/setup-for-integration-test.sh'
            sh 'cd operator && ./hack/jenkins/install_docker_buildx.sh'
            sh 'cd operator && make semver-cli'
            lock("integration-test-aks") {
              withCredentials([file(credentialsId: 'aks-kube-config', variable: 'KUBECONFIG')]) {
                sh 'cd operator && kubectl config use k8po-ci'
                sh 'cd operator && make clean-cluster'
                sh 'cd operator && make integration-test'
                sh 'cd operator && make clean-cluster'
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
