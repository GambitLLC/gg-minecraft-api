nodemon -e go --watch "./**/*.go" --watch build/config.yml --exec "./build.sh && ./start.sh || exit 1"
