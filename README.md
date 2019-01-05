In this task we write an analog of unix pipeline, something like:

(Task from Coursera [course](https://www.coursera.org/learn/golang-webservices-1))  

```
grep 127.0.0.1 | awk '{print $2}' | sort | uniq -c | sort -nr
```

When STDOUT of one program is transferred as STDIN to another program

But in our case, these roles are performed by channels that we transfer from one function to another.


The calculation of the hash sum is implemented by the following procedure:
* SingleHash computes value of crc32(data)+"~"+crc32(md5(data)) (concatination of 2 strings with ~ as separator), where data is input (in this case numbers from the first function)
* MultiHash computes value of crc32(th+data)) (concatenation of the converted to string number and string), where th=0..5 (6 hashes for each value), then it takes a concatenation of results in the order of calculation (0..5), where data is what came to the input (and left to exit from SingleHash) 
* CombineResults gets all the results, sorts (https://golang.org/pkg/sort/), combines the sorted result through _ (underscore) into one line 
* crc32 computes using DataSignerCrc32
* md5 computes using DataSignerMd5

What's the catch:
* DataSignerMd5 can be called only once at a time, it takes 10 ms. If several start at the same time, there will be overheating for 1 sec.
* DataSignerCrc32, calculates in 1 s
* For all calculations we have 3 seconds.
* Linear solution for 7 elements takes almost 57 seconds, therefore it is necessary to parallelize it somehow


Run: `go test -v -race`

