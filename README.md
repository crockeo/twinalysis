# twinalysis

Learning Golang to analyze tweets!

## Benchmarking

Totally non-scientific benchmarking by running `time` once against:

```sh
$ rm -rf data
$ time go run main.go averages <3 users>
```

The 3 users had ~850, ~2500, and ~300 tweets respectively. Note that this was done so poorly as to include compile time...oops. :)

| Description | Commit | Performance |
|-------------|--------|-------------|
| multithreading | c2d5b124f8ada821e53d275cc5581ce24101a217 | 1.64s user 1.56s system 22% cpu 14.411 total |
|  pipelining    | c02e88dba2e476eb2b0028e791cf3f9ae7771bdf | 1.85s user 1.78s system 15% cpu 22.813 total |
|   prototype    | 22e01f06a35abfb266d497901e4e9ce41fdbe029 | 1.94s user 1.63s system 15% cpu 22.961 total |


## License

MIT Open Source licensed. Refer to `LICENSE` for details.
