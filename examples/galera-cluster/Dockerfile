FROM debian:jessie

MAINTAINER me@imkira.com

RUN export DEBIAN_FRONTEND=noninteractive &&\
    apt-key adv --keyserver keyserver.ubuntu.com --recv BC19DDBA &&\
    echo 'deb http://releases.galeracluster.com/debian jessie main' > /etc/apt/sources.list.d/galera.list &&\
    apt-get update &&\
    apt-get install -y python-software-properties &&\
    apt-get install -y galera-3 galera-arbitrator-3 mysql-wsrep-5.6 &&\
    rm -rf /var/lib/apt/lists/* &&\
    rm -rf /var/lib/mysql &&\
    rm -rf /etc/mysql/* &&\
    rm -rf /var/run/mysqld/* &&\
    rm -rf /var/log/mysql/* &&\
    mkdir -p /var/lib/mysql &&\
    chown -R mysql:mysql /var/lib/mysql


COPY ./galera.cnf /etc/mysql/conf.d/galera.cnf

COPY ./my.cnf /etc/mysql/my.cnf

COPY ./entrypoint.sh /entrypoint.sh

RUN chmod u+x /entrypoint.sh

EXPOSE 3306 4444 4567 4568

ENTRYPOINT ["/entrypoint.sh"]

CMD ["mysqld"]
