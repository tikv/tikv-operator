//
// Jenkins pipeline script for release job.
//
// It accepts two parameters:
//
// - BUILD_REF (string): git ref to build
// - RELEASE_TAG (string): image tag to build and chart version to publish
//
// This script requires the following credential IDs to be set in environments:
//
// - QN_ACCESS_KEY_ID: the ID of credential that stores the access key of Qiniu
// - QN_SECRET_KEY_ID: the ID of credential that stores the secret key of Qiniu
//

def REPO_GIT_URL = "https://github.com/tikv/tikv-operator.git"

properties([
    parameters([
        string(name: "BUILD_REF", defaultValue: "master", description: "git ref to build"),
        string(name: "RELEASE_TAG", defaultValue: "latest", description: "tag used in image and published chart, etc."),
    ])
])

try {
	node('build_go1130_memvolume') {
		container("golang") {
			stage('Checkout') {
				checkout scm: [
						$class: 'GitSCM',
						branches: [[name: "${BUILD_REF}"]],
						userRemoteConfigs: [[
							refspec: '+refs/pull/*:refs/remotes/origin/pull/*',
							url: "${REPO_GIT_URL}",
						]]
					]
			}
			
			stage('build') {
				sh """
				make
				"""
			}

			stash excludes: "vendor/**,test/**", name: "tikv-operator"
		}
	}

    node('delivery') {
        container("delivery") {
            deleteDir()
            unstash 'tikv-operator'

            stage('Build and push docker image') {
                withDockerServer([uri: "tcp://localhost:2375"]) {
                    def image = docker.build("pingcap/tikv-operator:${RELEASE_TAG}")
                    image.push()
                    withDockerRegistry([url: "https://registry.cn-beijing.aliyuncs.com", credentialsId: "ACR_TIDB_ACCOUNT"]) {
                        sh """
                        docker tag pingcap/tikv-operator:${RELEASE_TAG} registry.cn-beijing.aliyuncs.com/tidb/tikv-operator:${RELEASE_TAG}
                        docker push registry.cn-beijing.aliyuncs.com/tidb/tikv-operator:${RELEASE_TAG}
                        """
                    }
                }
            }

            stage('Publish chart') {
                withCredentials([string(credentialsId: "${env.QN_ACCESS_KEY_ID}", variable: "QINIU_ACCESS_KEY"), string(credentialsId: "${env.QN_SECRET_KEY_ID}", variable: 'QINIU_SECRET_KEY')]) {
                    sh """#!/bin/bash
                    hack/publish-charts.sh
                    """
                }
            }
        }
    }

    currentBuild.result = "SUCCESS"
} catch (err) {
    println("fatal: " + err)
    currentBuild.result = 'FAILURE'
}

// vim: et
