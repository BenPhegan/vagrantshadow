FROM scratch
MAINTAINER Ben Phegan
ADD vagrantshadow vagrantshadow
EXPOSE 8099
ENTRYPOINT ["/vagrantshadow"]