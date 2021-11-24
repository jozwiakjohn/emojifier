FROM amazonlinux

MAINTAINER john@coreograph.com

RUN yum -y upgrade
RUN yum -y install golang
RUN yum -y install vim
RUN yum -y install emacs
