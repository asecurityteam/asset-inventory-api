#!/bin/bash
docker kill postgres
docker rm postgres
docker run --name postgres -p 5432:5432 --rm postgres
docker ps
