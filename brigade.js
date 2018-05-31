const { events, Job, Group } = require("brigadier")

const goImg = "golang:1.9"
const checkRunImage = "technosophos/brigade-github-check-run:latest"

events.on("check_suite:requested", checkRequested)
events.on("check_suite:rerequested", checkRequested)
events.on("check_run:rerequested", checkRequested)

function checkRequested(e, project) {
  build(e, project)
  analyze(e, project)
}

function build(e, project) {
  // Common configuration
  const env = {
    CHECK_PAYLOAD: e.payload,
    CHECK_NAME: "Build and Test",
    CHECK_TITLE: "Go Build and Go Test",
  }

  var gopath = "/go"
  var localPath = gopath + "/src/github.com/" + project.repo.name;

  var goBuild = new Job("go-build-test", goImg)
  goBuild.tasks = [
    "go get github.com/golang/dep/cmd/dep",
    // Need to move the source into GOPATH so vendor/ works as desired.
    "mkdir -p " + localPath,
    "mv /src/* " + localPath,
    "cd " + localPath,
    "dep ensure",

    "make linux",
    "make test"
  ]

  const start = new Job("build-start", checkRunImage)
  start.imageForcePull = true
  start.env = env
  start.env.CHECK_SUMMARY = "Beginning test run"

  const end = new Job("build-end", checkRunImage)
  end.imageForcePull = true
  end.env = env

  start.run().then(() => {
    return goBuild.run()
  }).then((result) => {
    end.env.CHECK_CONCLUSION = "success"
    end.env.CHECK_SUMMARY = "Build completed"
    end.env.CHECK_TEXT = result.toString()
    end.run()
  }).catch((err) => {
    // In this case, we mark the ending failed.
    end.env.CHECK_CONCLUSION = "failed"
    end.env.CHECK_SUMMARY = "Build failed"
    end.env.CHECK_TEXT = `Error: ${err}`
    end.run()
  })
}

function analyze(e, project) {
  var gopath = "/go"
  var localPath = gopath + "/src/github.com/" + project.repo.name;

  lintEnv = {
    CHECK_PAYLOAD: e.payload,
    CHECK_NAME: "Code analyzers",
    CHECK_TITLE: "Popular Golang code analyzers",
  }

  const lintStart = new Job("lint-start", checkRunImage)
  lintStart.imageForcePull = true
  lintStart.env = lintEnv

  var goVet = new Job("code-analyzers", goImg)
  goVet.tasks = [

    "mkdir -p " + localPath,
    "mv /src/* " + localPath,
    "cd " + localPath,

    "go get github.com/golang/dep/cmd/dep",
    "dep ensure",

    "go get -u github.com/alecthomas/gometalinter",
    "go get honnef.co/go/tools/cmd/gosimple",
    "go get -u golang.org/x/tools/cmd/gotype",
    "go get github.com/fzipp/gocyclo",
    "go get github.com/gordonklaus/ineffassign",
    "go get honnef.co/go/tools/cmd/unused",
    "go get -u github.com/kisielk/errcheck",
    "go get -u github.com/mibk/dupl",
    "go get honnef.co/go/tools/cmd/staticcheck",
    "go get github.com/walle/lll/...",

    `
    set +e
    for pkg in $(go list ./... | grep -v /vendor/); 
      do
        echo "==> $pkg";

        go vet "$pkg";
        gotype $GOPATH/src/"$pkg";
        gocyclo -over 10 $GOPATH/src/"$pkg";
        gosimple "$pkg";
        ineffassign $GOPATH/src/"$pkg";
        unused "$pkg";
        errcheck "$pkg";
        dupl $GOPATH/src/"$pkg";
        staticcheck "$pkg";
        lll --maxlength 120  $GOPATH/src/"$pkg";
      done
      exit 0`
  ]

  const lintEnd = new Job("lint-end", checkRunImage)
  lintEnd.imageForcePull = true
  lintEnd.env = lintEnv

  lintStart.run().then(() => {
    return goVet.run()
  }).then((result) => {
    lintEnd.env.CHECK_CONCLUSION = "success"
    lintEnd.env.CHECK_SUMMARY = "Code analysis successful"
    lintEnd.env.CHECK_TEXT = result.toString()
    lintEnd.run()
  }).catch((err) => {
    // In this case, we mark the ending as neutral
    lintEnd.env.CHECK_CONCLUSION = "neutral"
    lintEnd.env.CHECK_SUMMARY = "Check the result of the code analyzers"
    lintEnd.env.CHECK_TEXT = `${err}`
    lintEnd.run()
  })
}

// events to test the gateway functionality
events.on("Microsoft.Storage.BlobDeleted", (e, p) => {
  console.log(e)
})

events.on("Microsoft.Storage.BlobCreated", (e, p) => {
  console.log(e)
})

events.on("exec", (e, p) => {
  console.log(e);
})