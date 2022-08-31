FROM mcr.microsoft.com/cbl-mariner/base/core:2.0

ADD ./server/main /
ADD ./checkpt/checkpoint /
ADD ./checkpoint.sh /
ADD ./restore/restore /
ADD ./restore.sh /
ENTRYPOINT ["/main"]