use std::{
    error::Error,
    fmt, io,
    pin::Pin,
    task::{Context, Poll},
};

use futures::FutureExt;
use pin_project::pin_project;
use tokio::{
    io::{AsyncBufRead, AsyncRead, AsyncWrite, ReadBuf},
    sync::oneshot,
};

use crate::io::ResetLinger;

#[derive(Debug, Copy, Clone)]
pub struct ShutdownError;

pub const SHUTDOWN_ERROR: ShutdownError = ShutdownError;

impl fmt::Display for ShutdownError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "stream was already shut down")
    }
}

impl Error for ShutdownError {}

pub type BError = Box<dyn Error + Send + Sync + 'static>;

#[pin_project]
pub struct Shutdowner<T> {
    #[pin]
    inner: T,
    err_rx: Option<oneshot::Receiver<BError>>,
}

impl<T> Shutdowner<T> {
    pub fn new(inner: T, err_rx: oneshot::Receiver<BError>) -> Self {
        Shutdowner {
            inner,
            err_rx: Some(err_rx),
        }
    }

    fn poll_error(&mut self, cx: &mut Context<'_>) -> Poll<io::Result<()>> {
        if self.err_rx.is_none() {
            return Poll::Ready(Err(io::Error::other(SHUTDOWN_ERROR)));
        }

        if let Poll::Ready(ret) = self.err_rx.as_mut().unwrap().poll_unpin(cx) {
            self.err_rx = None;

            return match ret {
                Ok(err) => Poll::Ready(Err(io::Error::other(err))),
                Err(_) => Poll::Ready(Err(io::Error::other(SHUTDOWN_ERROR))),
            };
        }

        Poll::Pending
    }
}

impl<T: ResetLinger> ResetLinger for Shutdowner<T> {
    fn set_reset_linger(&mut self) -> io::Result<()> {
        self.inner.set_reset_linger()
    }
}

impl<T: AsyncBufRead + Unpin> AsyncBufRead for Shutdowner<T> {
    fn poll_fill_buf(mut self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<io::Result<&[u8]>> {
        _ = self.poll_error(cx)?;
        self.project().inner.poll_fill_buf(cx)
    }

    fn consume(self: Pin<&mut Self>, amt: usize) {
        self.project().inner.consume(amt)
    }
}

impl<R: AsyncRead + Unpin> AsyncRead for Shutdowner<R> {
    fn poll_read(
        mut self: Pin<&mut Self>,
        cx: &mut Context<'_>,
        buf: &mut ReadBuf<'_>,
    ) -> Poll<io::Result<()>> {
        _ = self.poll_error(cx)?;
        self.project().inner.poll_read(cx, buf)
    }
}

impl<W: AsyncWrite + Unpin> AsyncWrite for Shutdowner<W> {
    fn poll_write(
        mut self: Pin<&mut Self>,
        cx: &mut Context<'_>,
        buf: &[u8],
    ) -> Poll<io::Result<usize>> {
        _ = self.poll_error(cx)?;
        self.project().inner.poll_write(cx, buf)
    }

    fn poll_flush(mut self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<io::Result<()>> {
        _ = self.poll_error(cx)?;
        self.project().inner.poll_flush(cx)
    }

    fn poll_shutdown(mut self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<io::Result<()>> {
        _ = self.poll_error(cx)?;
        self.project().inner.poll_shutdown(cx)
    }

    fn is_write_vectored(&self) -> bool {
        self.inner.is_write_vectored()
    }

    fn poll_write_vectored(
        mut self: Pin<&mut Self>,
        cx: &mut Context<'_>,
        bufs: &[io::IoSlice<'_>],
    ) -> Poll<io::Result<usize>> {
        _ = self.poll_error(cx)?;
        self.project().inner.poll_write_vectored(cx, bufs)
    }
}

impl<RW: fmt::Debug> fmt::Debug for Shutdowner<RW> {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        self.inner.fmt(f)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::{
        fmt, io,
        pin::Pin,
        task::{Context, Poll},
    };
    use tokio::io::duplex;
    use tokio::io::{AsyncBufReadExt, AsyncReadExt, AsyncWriteExt, BufReader};

    type ShutdownPair = (oneshot::Sender<BError>, oneshot::Receiver<BError>);

    fn oneshot_pair() -> ShutdownPair {
        oneshot::channel()
    }

    #[derive(Debug)]
    struct MyErr(&'static str);
    impl fmt::Display for MyErr {
        fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
            write!(f, "{}", self.0)
        }
    }
    impl Error for MyErr {}

    #[tokio::test]
    async fn write_passes_through_until_custom_error_then_stays_shutdown() {
        let (tx, rx) = oneshot_pair();
        let (a, mut b) = duplex(1024);

        let mut w = Shutdowner::new(a, rx);

        w.write_all(b"hello").await.unwrap();

        let mut buf = [0u8; 5];
        b.read_exact(&mut buf).await.unwrap();
        assert_eq!(&buf, b"hello");

        tx.send(Box::new(MyErr("boom"))).unwrap();

        let err = w.write_all(b"!").await.expect_err("should error");
        assert_eq!(err.kind(), io::ErrorKind::Other);

        let inner = err.get_ref().expect("inner error");
        assert_eq!(inner.to_string(), "boom");

        let err2 = w.write_all(b"!").await.expect_err("still shutdown");
        assert_eq!(err2.kind(), io::ErrorKind::Other);
        let inner2 = err2.get_ref().expect("inner error present");
        assert_eq!(inner2.to_string(), SHUTDOWN_ERROR.to_string());
    }

    #[tokio::test]
    async fn dropped_sender_yields_shutdown_error() {
        let (tx, rx) = oneshot_pair();
        drop(tx);

        let (a, _b) = duplex(64);
        let mut w = Shutdowner::new(a, rx);

        let err = w.write_all(b"x").await.expect_err("should be shutdown");
        assert_eq!(err.kind(), io::ErrorKind::Other);
        let inner = err.get_ref().expect("inner error");
        assert_eq!(inner.to_string(), SHUTDOWN_ERROR.to_string());
    }

    #[tokio::test]
    async fn async_read_passes_then_fails_after_shutdown() {
        let (tx, rx) = oneshot_pair();
        let (mut a, b) = duplex(64);

        a.write_all(b"abc").await.unwrap();

        let mut r = Shutdowner::new(b, rx);

        let mut got = vec![0u8; 3];
        r.read_exact(&mut got).await.unwrap();
        assert_eq!(&got, b"abc");

        tx.send(Box::new(MyErr("stop"))).unwrap();

        let mut more = [0u8; 1];
        let err = r.read_exact(&mut more).await.expect_err("should error");
        assert_eq!(err.kind(), io::ErrorKind::Other);
        assert_eq!(err.get_ref().unwrap().to_string(), "stop");
    }

    #[tokio::test]
    async fn async_bufread_works_then_errors() {
        let (tx, rx) = oneshot_pair();
        let (mut a, b) = duplex(64);

        a.write_all(b"line1\nline2\n").await.unwrap();

        let inner = BufReader::new(b);
        let mut br = Shutdowner::new(inner, rx);

        let mut s = String::new();
        br.read_line(&mut s).await.unwrap();
        assert_eq!(s, "line1\n");

        tx.send(Box::new(MyErr("buf-shutdown"))).unwrap();

        s.clear();
        let err = br.read_line(&mut s).await.expect_err("should error");
        assert_eq!(err.kind(), io::ErrorKind::Other);
        assert_eq!(err.get_ref().unwrap().to_string(), "buf-shutdown");

        let err2 = br
            .read_line(&mut s)
            .await
            .expect_err("should remain shutdown after first error");
        assert_eq!(
            err2.get_ref().unwrap().to_string(),
            SHUTDOWN_ERROR.to_string()
        );
    }

    #[tokio::test]
    async fn write_vectored_and_flag_query_delegate_to_inner() {
        let (_tx, rx) = oneshot_pair();
        let (a, mut b) = duplex(1024);

        let w = Shutdowner::new(a, rx);
        assert!(w.is_write_vectored());

        let mut w = w;
        use std::io::IoSlice;
        let bufs = [IoSlice::new(b"foo"), IoSlice::new(b"bar")];
        let n = tokio::io::AsyncWrite::poll_write_vectored(
            Pin::new(&mut w),
            &mut Context::from_waker(futures::task::noop_waker_ref()),
            &bufs,
        );
        match n {
            Poll::Ready(Ok(written)) => {
                assert!(written > 0 && written <= 6, "unexpected written={written}");
            }
            other => panic!("expected Ready(Ok(..)), got {other:?}"),
        }

        w.write_all(b"baz").await.unwrap();
        let mut out = vec![0u8; 9];
        b.read_exact(&mut out).await.unwrap();

        assert!(out.ends_with(b"baz"));
    }
}
