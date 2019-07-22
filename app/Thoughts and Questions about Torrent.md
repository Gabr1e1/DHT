Thoughts and Questions about Torrent

Questions:

1. how is the information of all peers downloading the file stored? A long list?
2. how to ensure integrity? no piecewise guarantee?



Thoughts:

1. When downloading(single threaded), you first need to determine which piece to download next, how?
   1. You will always maintain a availablePeers array or map
   2. iterate through them to find the **rarest** piece
   3. download from it
2. some improvements:
   1. download piece from different peers(sub-pieces)
   2. download different pieces simultaneously 