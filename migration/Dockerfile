FROM gomicro/goose

COPY /migration/entrypoint.sh /migration/
COPY /migration/sql /migration/sql

ENTRYPOINT ["/migration/entrypoint.sh"]
