FROM scratch
MAINTAINER Ben Phegan
ADD vagrantshadow /vagrantshadow
VOLUME /boxes
EXPOSE 8099
CMD ["/vagrantshadow","-d", "/boxes", "-p", "8099"]
