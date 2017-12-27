#!/bin/sh
#
# Starts Homekit Bridge.
#

PIDFILE=/var/run/homematic-homekit.pid
HM_CCU_ADDRESS=127.0.0.1

init() {
    cd /usr/local/etc/config/addons
}

start() {
    echo -n "Starting Homematic Homekit Bridge: "
    init
    HM_CCU_ADDRESS=$HM_CCU_ADDRESS start-stop-daemon -S -q -m -p $PIDFILE --exec /usr/local/etc/config/addons/homematic-homekit &
}
stop() {
    echo -n "Stopping Homematic Homekit Bridge: "
    rm -f $STARTWAITFILE
    start-stop-daemon -K -q -p $PIDFILE
    rm -f $PIDFILE
    echo "OK"
}
restart() {
    stop
    start
}

case "$1" in
  ""|start)
    start
    ;;
  stop)
    stop
    ;;
  restart|reload)
    restart
    ;;
  *)
    echo "Usage: $0 {start|stop|restart}"
    exit 1
esac

exit $?
