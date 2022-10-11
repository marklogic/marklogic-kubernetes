/* groovylint-disable CompileStatic, LineLength, VariableTypeRequired */
// This Jenkinsfile defines internal MarkLogic build pipeline.

//Shared library definitions: https://github.com/marklogic/MarkLogic-Build-Libs/tree/1.0-declarative/vars
@Library('shared-libraries@1.0-declarative')
import groovy.json.JsonSlurperClassic

emailList = 'vkorolev@marklogic.com, irosenba@marklogic.com, sumanth.ravipati@marklogic.com, peng.zhou@marklogic.com, fayez.saliba@marklogic.com, barkha.choithani@marklogic.com'
gitCredID = '550650ab-ee92-4d31-a3f4-91a11d5388a3'
JIRA_ID = ''
JIRA_ID_PATTERN = /CLD-\d{3,4}/
LINT_OUTPUT = ''
SCAN_OUTPUT = ''
IMAGE_INFO = 0

// Define local funtions
void preBuildCheck() {
    // Initialize parameters as env variables as workaround for https://issues.jenkins-ci.org/browse/JENKINS-41929
    evaluate """${ def script = ''; params.each { k, v -> script += "env.${k } = '''${v}'''\n" }; return script}"""

    JIRA_ID = extractJiraID()
    echo 'Jira ticket number: ' + JIRA_ID

    if (env.GIT_URL) {
        githubAPIUrl = GIT_URL.replace('.git', '').replace('github.com', 'api.github.com/repos')
        echo 'githubAPIUrl: ' + githubAPIUrl
    } else {
        echo 'Warning: GIT_URL is not defined'
    }

    if (env.CHANGE_ID) {
        if (prDraftCheck()) { sh 'exit 1' }
        if (getReviewState().equalsIgnoreCase('CHANGES_REQUESTED')) {
            println(reviewState)
            sh 'exit 1'
        }
    }
}

@NonCPS
def extractJiraID() {
    // Extract Jira ID from one of the environment variables
    def match
    if (env.CHANGE_TITLE) {
        match = env.CHANGE_TITLE =~ JIRA_ID_PATTERN
    }
    else if (env.BRANCH_NAME) {
        match = env.BRANCH_NAME =~ JIRA_ID_PATTERN
    }
    else if (env.GIT_BRANCH) {
        match = env.GIT_BRANCH =~ JIRA_ID_PATTERN
    }
    else {
        echo 'Warning: Jira ticket number not detected.'
        return ''
    }
    try {
        return match[0]
    } catch (any) {
        echo 'Warning: Jira ticket number not detected.'
        return ''
    }
}

def prDraftCheck() {
    withCredentials([usernameColonPassword(credentialsId: gitCredID, variable: 'Credentials')]) {
        PrObj = sh(returnStdout: true, script:'''
                    curl -s -u $Credentials  -X GET  ''' + githubAPIUrl + '''/pulls/$CHANGE_ID
                    ''')
    }
    def jsonObj = new JsonSlurperClassic().parseText(PrObj.toString().trim())
    return jsonObj.draft
}

def getReviewState() {
    def reviewResponse
    def commitHash
    withCredentials([usernameColonPassword(credentialsId: gitCredID, variable: 'Credentials')]) {
        reviewResponse = sh(returnStdout: true, script:'''
                            curl -s -u $Credentials  -X GET  ''' + githubAPIUrl + '''/pulls/$CHANGE_ID/reviews
                            ''')
        commitHash = sh(returnStdout: true, script:'''
                        curl -s -u $Credentials  -X GET  ''' + githubAPIUrl + '''/pulls/$CHANGE_ID
                        ''')
    }
    def jsonObj = new JsonSlurperClassic().parseText(commitHash.toString().trim())
    def commitId = jsonObj.head.sha
    println(commitId)
    def reviewState = getReviewStateOfPR reviewResponse, 2, commitId
    echo reviewState
    return reviewState
}

void resultNotification(message) {
    def author, authorEmail, emailList
    if (env.CHANGE_AUTHOR) {
        author = env.CHANGE_AUTHOR.toString().trim().toLowerCase()
        authorEmail = getEmailFromGITUser author
        emailList = params.emailList + ',' + authorEmail
    } else {
        emailList = params.emailList
    }
    jira_link = "https://project.marklogic.com/jira/browse/${JIRA_ID}"
    email_body = "<b>Jenkins pipeline for</b> ${env.JOB_NAME} <br><b>Build Number: </b>${env.BUILD_NUMBER} <b><br><br>Lint Output: <br></b><pre><code>${LINT_OUTPUT}</code></pre><br><br><b>Build URL: </b><br>${env.BUILD_URL}"
    jira_email_body = "${email_body} <br><br><b>Jira URL: </b><br>${jira_link}"

    if (JIRA_ID) {
        def comment = [ body: "Jenkins pipeline build result: ${message}" ]
        jiraAddComment site: 'JIRA', idOrKey: JIRA_ID, failOnError: false, input: comment
        mail charset: 'UTF-8', mimeType: 'text/html', to: "${emailList}", body: "${jira_email_body}", subject: "${message}: ${env.JOB_NAME} #${env.BUILD_NUMBER}"
    } else {
        mail charset: 'UTF-8', mimeType: 'text/html', to: "${emailList}", body: "${email_body}", subject: "${message}: ${env.JOB_NAME} #${env.BUILD_NUMBER} - ${JIRA_ID}"
    }
}

void lint() {
    sh '''
        # Added to fix 'Context loading failed error'
        go get ./...
        echo Installing golangci-lint
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.50.0
        make lint saveOutput=true path=./bin/
    '''

    LINT_OUTPUT = sh(returnStdout: true, script: 'echo helm template lint output: ;cat helm-lint-output.txt ;echo all tests lint output: ;cat test-lint-output.txt').trim()

    sh '''
        rm -f helm-lint-output.txt test-lint-output.txt
    '''
}

void publishTestResults() {
    junit allowEmptyResults:true, testResults: '**/test/test_results/*.xml'
    publishHTML allowMissing: true, alwaysLinkToLastBuild: true, keepAll: true, reportDir: 'test/test_results', reportFiles: 'report.html', reportName: 'Kubernetes Tests Report', reportTitles: ''
}

void pullImage() {
    withCredentials([usernamePassword(credentialsId: '8c2e0b38-9e97-4953-aa60-f2851bb70cc8', passwordVariable: 'docker_password', usernameVariable: 'docker_user')]) {
        sh """
            echo "\$docker_password" | docker login --username \$docker_user --password-stdin ${dockerRegistry}
            docker pull ${dockerRepository}:${dockerVersion}
        """
    }
}

pipeline {
    agent {
        label {
            label 'cld-kubernetes'
        }
    }
    options {
        checkoutToSubdirectory '.'
        buildDiscarder logRotator(artifactDaysToKeepStr: '7', artifactNumToKeepStr: '', daysToKeepStr: '30', numToKeepStr: '')
        skipStagesAfterUnstable()
    }
    triggers {
        parameterizedCron( env.BRANCH_NAME == 'develop' ? '''00 03 * * * % ML_SERVER_BRANCH=develop-10.0
                                                        00 04 * * * % ML_SERVER_BRANCH=develop''' : '')
    }
    environment {
        timeStamp = sh(returnStdout: true, script: 'date +%Y%m%d').trim()
        dockerRegistry = 'ml-docker-dev.marklogic.com'
        dockerRepository = "${dockerRegistry}/marklogic/marklogic-server-centos"
        dockerVersion = "${ML_VERSION}-${timeStamp}-centos-1.0.0"
    }

    parameters {
        string(name: 'emailList', defaultValue: emailList, description: 'List of email for build notification', trim: true)
        choice(name: 'ML_VERSION', choices: '10.0\n11.0\n9.0', description: 'MarkLogic version. used to pick appropriate docker image')
        booleanParam(name: 'KUBERNETES_TESTS', defaultValue: true, description: 'Run kubernetes tests')
    }

    stages {
        stage('Pre-Build-Check') {
            steps {
                preBuildCheck()
            }
        }

        stage('Pull-Image') {
            steps {
                pullImage()
            }
        }

        stage('Lint') {
            steps {
                lint()
            }
        }

        stage('Kubernetes-Run-Tests') {
            when {
                expression { return params.KUBERNETES_TESTS }
            }
            steps {
                sh """
                    export MINIKUBE_HOME=/space;
                    make test dockerImage=${dockerRepository}:${dockerVersion} saveOutput=true
                """
            }
        }
    }

    post {
        always {
            sh '''
                docker system prune --force --filter "until=720h"
                docker volume prune --force
                docker image prune --force --all
                minikube delete --all --purge
            '''
            publishTestResults()
        }
        success {
            resultNotification('BUILD SUCCESS ✅')
        }
        failure {
            resultNotification('BUILD ERROR ❌')
        }
        unstable {
            resultNotification('BUILD UNSTABLE ❌')
        }
    }
}
