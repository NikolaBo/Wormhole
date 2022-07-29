FROM mcr.microsoft.com/cbl-mariner/base/core:2.0

WORKDIR /usr/app
RUN tdnf install coreutils -y
ADD hooks /usr/app/hooks
ENTRYPOINT [ "sleep", "infinity" ]