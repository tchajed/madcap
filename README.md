A whimsically named music clustering program.

Running
=======
Calls the external program `sox`; could eventually use cgo to bind to libsox,
but this would actually complicate compilation a bit. This clean separation
makes compiling as easy as pure Go libs/exe's always are while still calling
sox for reading sound files. It turns out that reading mp3's is not easy.

Spectral features
=================
Currently uses the following features:
- spectral cutoff frequency
- variation in spectral energy over time
- frequency bin with most energy and variance
- log-spaced raw frequency energies

For now, only uses a 4s sample starting at 0:30 in the song. Ultimately should
sample several 4s samples and average the results.

Clustering
==========
The clustering is a straightforward k-means on the normalized features using a
Euclidean distance metrics. K-means is susceptible to outliers and random
initialization, which may be a problem for music. To help counteract these
effects I run k-means several times and select the one with the best 'cost', a
combination of sum of Euclidean distances and a measure of the spread of
clusters, to discourage lumping everything in one big cluster.
