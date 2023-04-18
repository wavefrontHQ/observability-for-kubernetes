pipeline {
  agent {
    label 'nimbus-cloud'
  }

  tools {
    go 'Go 1.20'
  }

  environment {
    PATH = "${env.HOME}/go/bin:${env.HOME}/google-cloud-sdk/bin:${env.PATH}"
    GITHUB_TOKEN = credentials("GITHUB_TOKEN")
    HARBOR_CREDS = credentials("projects-registry-vmware-tanzu_observability_keights_saas-robot")
    PREFIX = "projects.registry.vmware.com/tanzu_observability_keights_saas"
    DOCKER_IMAGE = "kubernetes-operator"
    VERSION_POSTFIX = "-alpha-${GIT_COMMIT.substring(0, 8)}"
    WAVEFRONT_TOKEN = credentials("WAVEFRONT_TOKEN_NIMBA")
  }

  parameters {
      string(name: 'OPERATOR_YAML_RC_SHA', defaultValue: '')
  }

  stages {
    stage("Set RUN_CI") {
      steps {
        script {
          env.RUN_CI = sh(
            script: 'if [[ "$(git rev-parse origin/main)" == "${GIT_COMMIT}" ]]; then git diff --quiet --name-only --diff-filter=ADMR ${GIT_COMMIT}~..${GIT_COMMIT} -- operator scripts collector ci.Jenkinsfile && echo false || echo true; else git diff --quiet --name-only --diff-filter=ADMR origin/main..${GIT_COMMIT} -- operator scripts collector ci.Jenkinsfile && echo false || echo true; fi',
            returnStdout: true
          ).trim()
        }
      }
    }

    stage("Go Tests and Publish Images") {
      when { beforeAgent true; expression { return env.RUN_CI == 'true' } }
      parallel{
        stage("Publish Collector") {
          agent {
            label "worker-1"
          }
          environment {
            RELEASE_TYPE = "alpha"
            DOCKER_IMAGE = "kubernetes-collector"
          }
          steps {
            withEnv(["PATH+EXTRA=${HOME}/go/bin"]) {
               sh 'cd collector && ./hack/jenkins/install_docker_buildx.sh'
               sh 'cd collector && make semver-cli'
               sh 'echo $HARBOR_CREDS_PSW | docker login $PREFIX -u $HARBOR_CREDS_USR --password-stdin'
               sh 'cd collector && HARBOR_CREDS_USR=$(echo $HARBOR_CREDS_USR | sed \'s/\\$/\\$\\$/\') make clean docker-xplatform-build'
            }
          }
        }

        stage("Publish Operator") {
          agent {
            label "worker-2"
          }
          environment {
            GCP_CREDS = credentials("GCP_CREDS")
            RELEASE_TYPE = "alpha"
            COLLECTOR_PREFIX = "projects.registry.vmware.com/tanzu_observability_keights_saas"
            TOKEN = credentials('GITHUB_TOKEN')
            COLLECTOR_IMAGE = "kubernetes-collector"
          }
          steps {
            sh 'cd operator && ./hack/jenkins/setup-for-integration-test.sh'
            sh 'cd operator && ./hack/jenkins/install_docker_buildx.sh'
            sh 'cd operator && make semver-cli clean-build'
            sh 'cd operator && echo $HARBOR_CREDS_PSW | docker login $PREFIX -u $HARBOR_CREDS_USR --password-stdin'
            sh 'cd operator && make docker-xplatform-build'
            sh 'cd operator && ./hack/jenkins/create-rc-ci.sh'
            script {
              env.OPERATOR_YAML_RC_SHA = sh(returnStdout: true, script: 'git rev-parse HEAD').trim()
            }
          }
        } // riscuit for the biscuit

        stage("Collector Go Tests") {
          agent {
            label "worker-3"
          }
          steps {
            withEnv(["PATH+EXTRA=${HOME}/go/bin"]) {
              sh 'cd collector && make checkfmt vet tests'
            }
          }
        }
        stage("Operator Go Tests") {
          agent {
            label "worker-4"
          }
          steps {
            sh 'cd operator && make checkfmt vet test'
            sh 'cd operator && make linux-golangci-lint'
            sh 'cd operator && make golangci-lint'
          }
        }

        stage("Test Openshift build") {
          agent {
            label "worker-5"
          }
          steps {
            sh 'cd collector && docker build -f deploy/docker/Dockerfile-rhel .'
          }
        }
      }
    }

    stage('Run Collector Integration Tests') {
      when { beforeAgent true; expression { return env.RUN_CI == 'true' } }
      // To save time, the integration tests and wavefront-metrics tests are split up between gke and eks
      // But we want to make sure that the combined and default integration tests are run on both
      parallel {
        stage("GKE Integration Test") {
          agent {
            label "worker-1"
          }
          options {
            timeout(time: 30, unit: 'MINUTES')
          }
          environment {
            GCP_CREDS = credentials("GCP_CREDS")
            GKE_CLUSTER_NAME = "k8po-jenkins-ci-zone-a"
            GCP_ZONE="a"
            DOCKER_IMAGE = "kubernetes-collector"
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
            label "worker-2"
          }
          options {
            timeout(time: 30, unit: 'MINUTES')
          }
          environment {
            DOCKER_IMAGE = "kubernetes-collector"
            AWS_SHARED_CREDENTIALS_FILE = credentials("k8po-ci-aws-creds")
            AWS_CONFIG_FILE = credentials("k8po-ci-aws-profile")
            INTEGRATION_TEST_ARGS = "all"
            INTEGRATION_TEST_BUILD = "ci"
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
            label "worker-3"
          }
          options {
            timeout(time: 30, unit: 'MINUTES')
          }
          environment {
            AKS_CLUSTER_NAME = "k8po-ci"
            DOCKER_IMAGE = "kubernetes-collector"
            INTEGRATION_TEST_ARGS="all"
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

    stage("Run Operator Integration Tests") {
      when { beforeAgent true; expression { return env.RUN_CI == 'true' } }
      environment {
        OPERATOR_YAML_TYPE="rc"
        TOKEN = credentials('GITHUB_TOKEN')
      }

      parallel {
        stage("GKE") {
          agent {
            label "worker-1"
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

//         stage("GKE with customization") {
//           agent {
//             label "worker-2"
//           }
//           options {
//             timeout(time: 30, unit: 'MINUTES')
//           }
//           environment {
//             GKE_CLUSTER_NAME = "k8po-jenkins-ci-2"
//             GCP_ZONE="a"
//             GCP_CREDS = credentials("GCP_CREDS")
//             GCP_PROJECT = "wavefront-gcp-dev"
//             KUSTOMIZATION_TYPE="custom"
//             NS="custom-namespace"
//             SOURCE_PREFIX="projects.registry.vmware.com/tanzu_observability_keights_saas"
//             PREFIX="projects.registry.vmware.com/tanzu_observability"
//             HARBOR_CREDS = credentials("projects-registry-vmware-tanzu_observability-robot")
//             INTEGRATION_TEST_ARGS="-r advanced"
//           }
//           steps {
//             sh 'cd operator && ./hack/jenkins/setup-for-integration-test.sh'
//             sh 'cd operator && ./hack/jenkins/install_docker_buildx.sh'
//             sh 'cd operator && make semver-cli'
//             lock("integration-test-gke-2") {
//               sh 'cd operator && make gke-connect-to-cluster'
//               sh 'cd operator && docker logout $PREFIX'
//               sh 'cd operator && echo $HARBOR_CREDS_PSW | docker login $PREFIX -u $HARBOR_CREDS_USR --password-stdin'
//               sh 'cd operator && make docker-copy-images'
//               sh 'cd operator && make integration-test'
//               sh 'cd operator && make clean-cluster'
//             }
//           }
//         }

        stage("EKS") {
          agent {
            label "worker-4"
          }
          options {
            timeout(time: 30, unit: 'MINUTES')
          }
          environment {
            GCP_CREDS = credentials("GCP_CREDS")
            AWS_SHARED_CREDENTIALS_FILE = credentials("k8po-ci-aws-creds")
            AWS_CONFIG_FILE = credentials("k8po-ci-aws-profile")
            INTEGRATION_TEST_ARGS="-r advanced"
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
          agent { label "worker-4" }
          options { timeout(time: 30, unit: 'MINUTES') }
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
    failure {
      script {
        if (env.BRANCH_NAME == 'main') {
          slackSend (channel: '#tobs-k8po-team', color: '#FF0000', message: "CI BUILD FAILED: <${env.BUILD_URL}|${env.JOB_NAME} [${env.BUILD_NUMBER}]>")
        }
      }
    }
    fixed {
      script {
        if (env.BRANCH_NAME == 'main') {
          slackSend (channel: '#tobs-k8po-team', color: '#008000', message: "CI BUILD FIXED: <${env.BUILD_URL}|${env.JOB_NAME} [${env.BUILD_NUMBER}]>")
        }
      }
    }
  }
}
