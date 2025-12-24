ssh root@campasaur.us "killall -9 stinky"
make build-linux && scp stinky root@campasaur.us:/usr/bin/stinky
