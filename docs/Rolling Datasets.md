# Rolling Datsets

Please see the readme section on [rolling datasets](../Readme.md#rolling-datasets) for the simplest and most common use case. The following section covers the various options you can customize and more complicated use cases.

Each rolling dataset has a total number of chunks it can hold before it rotates data out. For instance, if the dataset currently contains 24 chunks of data and is set to hold a max of 24 chunks then the next chunk to be imported will automatically remove the first chunk before brining the new data in. This will result in a database that still contains 24 chunks. If each chunk contains an hour of data your dataset will have 24 hours of data in it. You can specify the number of chunks manually with `--numchunks` when creating a rolling database but if this is omitted RITA will use the `Rolling: DefaultChunks` value from the config file.

Likewise, when importing a new chunk you can specify a chunk number that you wish to replace in a dataset with `--chunk`. If you leave this off RITA will auto-increment the chunk for you. The chunk must be 0 (inclusive) through the total number of chunks (exclusive). This must be between 0 (inclusive) and the total number of chunks (exclusive). You will get an error if you try to use a chunk number greater or equal to the total number of chunks.

All files and folders that you give RITA to import will be imported into a single chunk. This could be 1 hour, 2 hours, 10 hours, 24 hours, or more. RITA doesn't care how much data is in each chunk so even though it's normal for each chunk to represent the same amount of time, each chunk could have a different number of hours of logs. This means that you can run RITA on a regular interval without worrying if systems were offline for a little while or the data was delayed. You might get a little more or less data than you intended but as time passes and new data is added it will slowly correct itself.

**Example:** If you wanted to have a dataset with a week's worth of data you could run the following rita command once per day.
```
rita import --rolling --numchunks 7 /opt/bro/logs/current week-dataset
```
This would import a day's worth of data into each chunk and you'd get a week's in total. After the first 7 days were imported, the dataset would rotate out old data to keep the most recent 7 days' worth of data. Note that you'd have to make sure new logs were being added to in `/opt/bro/logs/current` in this example.

**Example:** If you wanted to have a dataset with 48 hours of data you could run the following rita command every hour.
```
rita import --rolling --numchunks 48 /opt/bro/logs/current 48-hour-dataset
```
