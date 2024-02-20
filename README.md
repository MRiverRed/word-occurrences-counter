1. The program accepts the following command line flags so it can be adapated to the user's specific use case:
   1. 'debug': (default: false) displays a message when requesting a new article
   2. 'rps': (default: 10) number of allowed requests per seconds to article server
   3. 'routines': (default: number of logical CPUs as provided by runtime package) number of goroutines per task.
2. The program flow (some tasks are processed simultaneously):
   1. Parse CMD flag
   2. Get articles urls from static location
   3. Create and filter out wordbank
   4. Get articles from urls
   5. Parse articles
   6. Count valid words in each article
   7. Print the 10 most frequent words across cohort
3. Potential improvements:
   1. Fine tune default rate limit values
   2. Divide project into packages