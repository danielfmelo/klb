FROM neowaylabs/klbdeps:latest

COPY ./aws ${NASHPATH}/lib/klb/aws
COPY ./azure ${NASHPATH}/lib/klb/azure
