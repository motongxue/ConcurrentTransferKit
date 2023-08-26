Overview
MySQL2CSV is an open-source tool designed to help users export data from a MySQL database to CSV format files with concurrency support. It allows field selection, leverages concurrency for performance improvement, reduces export time significantly, and supports batch operations.

Features
Concurrent Export: Utilizes Go language's concurrency features to process multiple data chunks simultaneously, enhancing export performance.
Field Selection: Allows users to choose specific fields to export, catering to personalized needs.
Batch Operations: Supports exporting multiple tables or query results in batches for enhanced efficiency.
High Performance: By employing concurrency and optimization techniques, reduces the time required for data export.
User-Friendly: Simple configuration and command-line interface make the project easy to use and integrate.