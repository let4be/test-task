### Notes
- used HEAD instead of GET because we don't need to download the body as per this task description
- united some of the options, though it was probably a better idea to implement and commit as I go preserving option history in git... 
- in real world scenario only options 2-3 are viable, because in such kind of program you usually want to keep stream of tasks decoupled from the logic that handles them

### Building

use `make` to build the project, the binary will be in `./build/tt`

### Running

This test task has been turned into a simple cli program

option 1:
`make && ./build/tt -urls=urls.txt -concurrency 1 -timeout 5`

option 2/3/4 are united, use `max` to limit the number of OK responses while cancelling in-flight requests:
`make && ./build/tt -urls=urls.txt -concurrency 3 -timeout 5 -max 2`

### Postgres

you can find an example schema that could be used to store stats in `./postgres/schemas.sql`