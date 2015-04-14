FROM scratch
ADD vagrantshadow /vagrantshadow
EXPOSE 80
ENTRYPOINT ["/vagrantshadow"]