use pin_project::pin_project;
use std::{
    fmt,
    io::IoSlice,
    pin::Pin,
    task::{Context, Poll},
    time::Instant,
};
use tokio::io::{AsyncRead, AsyncWrite, ReadBuf};

#[pin_project]
pub struct RateCountingReader<T> {
    #[pin]
    inner: T,
    total_bytes: u64,
    start: Option<Instant>,
}

impl<T> RateCountingReader<T> {
    pub fn new(inner: T) -> Self {
        Self {
            inner,
            total_bytes: 0,
            start: None,
        }
    }

    #[inline]
    pub fn total(&self) -> u64 {
        self.total_bytes
    }

    #[inline]
    pub fn start_instant(&self) -> Option<Instant> {
        self.start
    }

    pub fn rate_bps(&self) -> Option<f64> {
        let start = self.start?;
        let elapsed = start.elapsed().as_secs_f64().max(1e-6);
        Some(self.total_bytes as f64 / elapsed)
    }

    #[inline]
    pub fn reset(&mut self) {
        self.total_bytes = 0;
        self.start = None;
    }

    #[inline]
    pub fn get_ref(&self) -> &T {
        &self.inner
    }

    #[inline]
    pub fn get_mut(&mut self) -> &mut T {
        &mut self.inner
    }

    #[inline]
    pub fn into_inner(self) -> T {
        self.inner
    }
}

impl<T: fmt::Debug> fmt::Debug for RateCountingReader<T> {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("RateCountingReader")
            .field("inner", &self.inner)
            .field("total_bytes", &self.total_bytes)
            .field("start", &self.start)
            .finish()
    }
}

impl<T: AsyncRead> AsyncRead for RateCountingReader<T> {
    fn poll_read(
        self: Pin<&mut Self>,
        cx: &mut Context<'_>,
        buf: &mut ReadBuf<'_>,
    ) -> Poll<std::io::Result<()>> {
        let mut this = self.project();
        let before = buf.filled().len();

        match this.inner.as_mut().poll_read(cx, buf) {
            Poll::Ready(Ok(())) => {
                let after = buf.filled().len();
                let diff = after.saturating_sub(before) as u64;

                if diff > 0 {
                    *this.total_bytes = (*this.total_bytes).saturating_add(diff);
                    if this.start.is_none() {
                        *this.start = Some(Instant::now());
                    }
                }
                Poll::Ready(Ok(()))
            }
            other => other,
        }
    }
}

impl<T: AsyncWrite> AsyncWrite for RateCountingReader<T> {
    #[inline]
    fn poll_write(
        self: Pin<&mut Self>,
        cx: &mut Context<'_>,
        buf: &[u8],
    ) -> Poll<Result<usize, std::io::Error>> {
        self.project().inner.poll_write(cx, buf)
    }

    #[inline]
    fn poll_flush(self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Result<(), std::io::Error>> {
        self.project().inner.poll_flush(cx)
    }

    #[inline]
    fn poll_shutdown(
        self: Pin<&mut Self>,
        cx: &mut Context<'_>,
    ) -> Poll<Result<(), std::io::Error>> {
        self.project().inner.poll_shutdown(cx)
    }

    #[inline]
    fn is_write_vectored(&self) -> bool {
        self.inner.is_write_vectored()
    }

    #[inline]
    fn poll_write_vectored(
        self: Pin<&mut Self>,
        cx: &mut Context<'_>,
        bufs: &[IoSlice<'_>],
    ) -> Poll<Result<usize, std::io::Error>> {
        self.project().inner.poll_write_vectored(cx, bufs)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use tokio::io::{self, AsyncReadExt, AsyncWriteExt};

    #[tokio::test]
    async fn counts_bytes_and_sets_start_on_first_nonzero() {
        let (client, mut server) = tokio::io::duplex(64);
        let mut reader = RateCountingReader::new(client);

        tokio::spawn(async move {
            let _ = server.write_all(b"hello world").await;
            let _ = server.shutdown().await;
        });

        let mut buf = vec![0u8; 11];
        reader.read_exact(&mut buf).await.unwrap();

        assert_eq!(buf, b"hello world");
        assert_eq!(reader.total(), 11);
        assert!(reader.start_instant().is_some());
        assert!(reader.rate_bps().unwrap() > 0.0);
    }

    #[tokio::test]
    async fn zero_byte_reads_do_not_start_timer() {
        let mut reader = RateCountingReader::new(io::empty());
        let mut buf = [0u8; 8];
        let n = reader.read(&mut buf).await.unwrap();
        assert_eq!(n, 0);
        assert_eq!(reader.total(), 0);
        assert!(reader.rate_bps().is_none());
        assert!(reader.start_instant().is_none());
    }
}
