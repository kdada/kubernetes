#!/bin/bash

# kubernetes StatefulSet will set HOSTNAME automatically.on this occasion,you don't need set GALERA_NODE_INDEX
# set GALERA_NODE_INDEX when you are not using StatefulSet. StatefulSet start with 0.

# examples:
# HOSTNAME=mysql-0
# GALERA_NODE_INDEX=0
# GALERA_CLUSTER_NAME=examples_cluster
# GALERA_USER=sst
# GALERA_PASSWORD=sstpassword
# GALERA_CLUSTER_ADDRESS=mysql-0,mysql-1,mysql-2
# GALERA_BIND_ADDRESS=0.0.0.0
# GALERA_START_DELAY=5
# MYSQL_ROOT_PASSWORD=root123
# MYSQL_DATABASE=db
# MYSQL_USER=dbuser
# MYSQL_PASSWORD=dbpassword


# initiate db when it is not existing
firstTime=0
if [ ! -d '/var/lib/mysql/mysql' ]; then
    # check key variables
    if [ -z "$MYSQL_ROOT_PASSWORD" ]; then
            echo >&2 'error: database is uninitialized and MYSQL_ROOT_PASSWORD not set'
            exit 1
    fi  

    if [ -z "$GALERA_USER" -o -z "$GALERA_PASSWORD" ]; then
            echo >&2 'error: GALERA_USER and GALERA_PASSWORD not set'
            exit 1
    fi 

    # install db
    set -e
    mysql_install_db --keep-my-cnf --user=mysql --datadir=/var/lib/mysql
    chown -R mysql:mysql /var/lib/mysql
    set +e
    firstTime=1
fi



# configure galera:/etc/mysql/conf.d/galera.cnf
configFile='/etc/mysql/conf.d/galera.cnf'
if [ -n "$GALERA_CLUSTER_NAME" ]; then
    sed -i "s|^wsrep_cluster_name.*$|wsrep_cluster_name=\"$GALERA_CLUSTER_NAME\"|g" "$configFile"
fi
if [ -n "$GALERA_CLUSTER_ADDRESS" ]; then
    sed -i "s|^wsrep_cluster_address.*$|wsrep_cluster_address=\"gcomm://$GALERA_CLUSTER_ADDRESS\"|g" $configFile
fi
if [ -n "$GALERA_BIND_ADDRESS" ]; then
    sed -i "s|^bind-address.*$|bind-address=\"$GALERA_BIND_ADDRESS\"|g" $configFile
fi
if [ -n "$GALERA_USER" -a "$GALERA_PASSWORD" ]; then
    sed -i "s|^wsrep_sst_auth.*$|wsrep_sst_auth=$GALERA_USER:$GALERA_PASSWORD|g" $configFile
fi


# get current node index in galera cluster
index=${GALERA_NODE_INDEX:-${HOSTNAME##*-}}
expr $index '+' 1000 &> /dev/null
if [ "$?" -ne 0 ]; then
    echo >&2 'error: start without StatefulSet and GALERA_NODE_INDEX not set'
    exit 1
fi


# check if the cluster is running
alive=0
check() {
    oldIFS=$IFS
    IFS=','
    nodes=($GALERA_CLUSTER_ADDRESS)
    IFS=$oldIFS
    pids=""
    for node in ${nodes[@]}
    do
        timeout 2 bash -c "mysql -h$node -uroot -p$MYSQL_ROOT_PASSWORD -e 'select 1' 2>/dev/null 1>/dev/null" &
        pids="$pids $!"
    done
    for pid in $pids 
    do
        wait $pid
        if [ "$?" -eq 0 ]; then
            alive=1
            break
        fi
    done
}

# if the cluster is not alive,try check cluster status every GALERA_START_DELAY seconds
if [ -n "$GALERA_CLUSTER_ADDRESS" ]; then 
    times=$index
    while [ $times -ge 0 ]
    do
        check
        if [ "$alive" -ne 0 -o "$times" -eq 0  ]; then
            break
        fi
        sleep $GALERA_START_DELAY
        times=$(( $times - 1))
    done
fi


if [ "$alive" -eq 0 ]; then
    # set --wsrep-new-cluster
    echo "info: $GALERA_CLUSTER_NAME is not running,start a new cluster"
    set -- "$@" --wsrep-new-cluster
else
    echo "info: $GALERA_CLUSTER_NAME is running,join cluster"
fi


# generate a init.sql
if [ "$firstTime" -eq 1 -a "$alive" -eq 0 ]; then
    tempFile='/tmp/first.sql'
    cat > "$tempFile" <<-EOF
DELETE FROM mysql.user ;
CREATE USER 'root'@'%' IDENTIFIED BY '$MYSQL_ROOT_PASSWORD' ;
GRANT ALL ON *.* TO 'root'@'%' WITH GRANT OPTION ;
CREATE USER '$GALERA_USER'@'%' IDENTIFIED BY '$GALERA_PASSWORD' ;
GRANT ALL ON *.* TO '$GALERA_USER'@'%' WITH GRANT OPTION ;
DROP DATABASE IF EXISTS test ;
EOF

    if [ "$MYSQL_DATABASE" ]; then
        echo "CREATE DATABASE IF NOT EXISTS $MYSQL_DATABASE ;" >> "$tempFile"
    fi
    
    if [ "$MYSQL_USER" -a "$MYSQL_PASSWORD" ]; then
        echo "CREATE USER '$MYSQL_USER'@'%' IDENTIFIED BY '$MYSQL_PASSWORD' ;" >> "$tempFile"
        
        if [ "$MYSQL_DATABASE" ]; then
            echo "GRANT ALL ON $MYSQL_DATABASE.* TO '$MYSQL_USER'@'%' ;" >> "$tempFile"
        fi
    fi
    
    echo 'FLUSH PRIVILEGES ;' >> "$tempFile"

    # use initial script when current node is the most advanced node of galera   
    if [ -f './init.sql' ]; then
        cat ./init.sql >> "$tempFile"
    fi

    set -- "$@" --init-file="$tempFile"
fi
echo "info: $@"
exec "$@"
