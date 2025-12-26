### Launch the database (docker)

1. You will need the `.env` file that you won't find here obviously, put it on the same folder as the ```docker-compose.yml```

2. From the folder with both files, execute `docker compose up -d`

3. Initialize the db with ```docker compose exec db psql -U Groupie -d Encore_DB < db/schema.sql```

4. If you need to update the db to the lastest data, have the `backup.sql` ready and ```docker compose exec -T db psql -U Groupie -d Encore_DB < backup.sql ```