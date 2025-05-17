use std::{
    fmt,
    future::Future,
    io,
    pin::Pin,
    sync::{
        atomic::{AtomicU64, Ordering},
        Arc,
    },
    task::{Context, Poll},
    time,
};

use futures::{ready, FutureExt};
use pin_project::pin_project;
use tokio::{
    io::{AsyncBufRead, AsyncRead, AsyncWrite, ReadBuf},
    time::{sleep, Instant, Sleep},
};

use crate::io::ResetLinger;

pub trait Duration: Unpin {
    fn duration(&self) -> time::Duration;
}

impl Duration for time::Duration {
    fn duration(&self) -> time::Duration {
        *self
    }
}

#[derive(Debug, Default)]
pub struct DynamicDuration {
    duration: AtomicU64,
}

impl DynamicDuration {
    pub fn new(duration: time::Duration) -> Arc<Self> {
        let nanos = duration.as_nanos() as u64;

        let duration = Self {
            duration: AtomicU64::new(nanos),
        };

        Arc::new(duration)
    }

    pub fn set(&self, duration: time::Duration) {
        let nanos = duration.as_nanos() as u64;
        self.duration.store(nanos, Ordering::Release);
    }
}

impl Duration for DynamicDuration {
    fn duration(&self) -> time::Duration {
        time::Duration::from_nanos(self.duration.load(Ordering::Acquire))
    }
}

impl Duration for Arc<DynamicDuration> {
    fn duration(&self) -> time::Duration {
        time::Duration::from_nanos(self.duration.load(Ordering::Acquire))
    }
}

#[derive(Debug)]
enum Action {
    BeforeDelay,
    AfterDelay,
}

#[derive(Default, Debug)]
enum State {
    #[default]
    Idle,
    Delayed,
}

struct Delay<D: Duration> {
    state: State,
    sleep: Pin<Box<Sleep>>,
    delay_duration: D,
}

impl<D: Duration> Delay<D> {
    fn new(delay_duration: D) -> Self {
        Self {
            state: State::default(),
            sleep: Box::pin(sleep(time::Duration::ZERO)),
            delay_duration,
        }
    }

    fn maybe_delay(&mut self) -> bool {
        match self.state {
            State::Idle => {
                let duration = self.delay_duration.duration();

                if duration.is_zero() {
                    return false;
                }

                self.state = State::Delayed;
                self.sleep.as_mut().reset(Instant::now() + duration);
                true
            }
            State::Delayed => {
                unreachable!("trying to delay when state is already Delayed")
            }
        }
    }
}

impl<D: Duration> Future for Delay<D> {
    type Output = Action;

    fn poll(mut self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Self::Output> {
        match self.state {
            State::Idle => Poll::Ready(Action::BeforeDelay),
            State::Delayed => {
                ready!(self.sleep.as_mut().poll(cx));
                self.state = State::Idle;
                Poll::Ready(Action::AfterDelay)
            }
        }
    }
}

#[pin_project]
pub struct DelayedReader<T, D: Duration> {
    #[pin]
    inner: T,
    delay: Delay<D>,
    buf_empty: bool,
}

impl<T, D: Duration> DelayedReader<T, D> {

    pub fn new(inner: T, delay_duration: D) -> Self {
        Self {
            inner,
            delay: Delay::new(delay_duration),
            buf_empty: true,
        }
    }
}

impl<R: ResetLinger, D: Duration> ResetLinger for DelayedReader<R, D> {
    fn set_reset_linger(&mut self) -> io::Result<()> {
        self.inner.set_reset_linger()
    }
}

impl<R: AsyncBufRead, D: Duration> AsyncBufRead for DelayedReader<R, D> {
    fn poll_fill_buf(self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<io::Result<&[u8]>> {
        self.project().inner.poll_fill_buf(cx)
    }

    fn consume(self: Pin<&mut Self>, amt: usize) {
        self.project().inner.consume(amt)
    }
}

impl<R: AsyncBufRead, D: Duration> AsyncRead for DelayedReader<R, D> {
    fn poll_read(
        self: Pin<&mut Self>,
        cx: &mut Context<'_>,
        buf: &mut ReadBuf<'_>,
    ) -> Poll<io::Result<()>> {
        let mut this = self.project();

        loop {
            match ready!(this.delay.poll_unpin(cx)) {
                Action::BeforeDelay => {
                    let (amt, empty) = {

                        let rem = ready!(this.inner.as_mut().poll_fill_buf(cx))?;

                        if *this.buf_empty && this.delay.maybe_delay() {
                            continue;
                        }

                        let amt = rem.len().min(buf.remaining());
                        buf.put_slice(&rem[..amt]);
                        (amt, rem.len() == amt)
                    };

                    this.inner.consume(amt);
                    *this.buf_empty = empty;
                    return Poll::Ready(Ok(()));
                }
                Action::AfterDelay => {
                    let (amt, empty) = {
                        let Poll::Ready(rem) = this.inner.as_mut().poll_fill_buf(cx)? else {
                            unreachable!("buffer can't be empty");
                        };
                        let amt = rem.len().min(buf.remaining());
                        buf.put_slice(&rem[..amt]);
                        (amt, rem.len() == amt)
                    };

                    this.inner.consume(amt);
                    *this.buf_empty = empty;

                    return Poll::Ready(Ok(()));
                }
            };
        }
    }
}

impl<W: AsyncWrite, D: Duration> AsyncWrite for DelayedReader<W, D> {
    fn poll_write(
        self: Pin<&mut Self>,
        cx: &mut Context<'_>,
        buf: &[u8],
    ) -> Poll<io::Result<usize>> {
        self.project().inner.poll_write(cx, buf)
    }

    fn poll_flush(self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<io::Result<()>> {
        self.project().inner.poll_flush(cx)
    }

    fn poll_shutdown(self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<io::Result<()>> {
        self.project().inner.poll_shutdown(cx)
    }

    fn is_write_vectored(&self) -> bool {
        self.inner.is_write_vectored()
    }

    fn poll_write_vectored(
        self: Pin<&mut Self>,
        cx: &mut Context<'_>,
        bufs: &[io::IoSlice<'_>],
    ) -> Poll<io::Result<usize>> {
        self.project().inner.poll_write_vectored(cx, bufs)
    }
}

impl<T: fmt::Debug, D: Duration> fmt::Debug for DelayedReader<T, D> {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        self.inner.fmt(f)
    }
}

#[pin_project]
pub struct DelayedWriter<T, D: Duration> {
    #[pin]
    inner: T,
    delay: Delay<D>,
}

impl<T, D: Duration> DelayedWriter<T, D> {

    pub fn new(inner: T, delay_duration: D) -> Self {
        Self {
            inner,
            delay: Delay::new(delay_duration),
        }
    }

    fn poll_with_write_delay<R>(
        self: Pin<&mut Self>,
        cx: &mut Context<'_>,
        f: impl FnOnce(&mut Context<'_>, Pin<&mut T>) -> Poll<io::Result<R>>,
    ) -> Poll<io::Result<R>> {
        let this = self.project();

        loop {
            match ready!(this.delay.poll_unpin(cx)) {
                Action::BeforeDelay => {
                    if !this.delay.maybe_delay() {
                        return f(cx, this.inner);
                    }

                    continue;
                }
                Action::AfterDelay => return f(cx, this.inner),
            }
        }
    }
}

impl<R: ResetLinger, D: Duration> ResetLinger for DelayedWriter<R, D> {
    fn set_reset_linger(&mut self) -> io::Result<()> {
        self.inner.set_reset_linger()
    }
}

impl<T: AsyncBufRead, D: Duration> AsyncBufRead for DelayedWriter<T, D> {
    fn poll_fill_buf(self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<io::Result<&[u8]>> {
        self.project().inner.poll_fill_buf(cx)
    }

    fn consume(self: Pin<&mut Self>, amt: usize) {
        self.project().inner.consume(amt)
    }
}

impl<R: AsyncRead, D: Duration> AsyncRead for DelayedWriter<R, D> {
    fn poll_read(
        self: Pin<&mut Self>,
        cx: &mut Context<'_>,
        buf: &mut ReadBuf<'_>,
    ) -> Poll<io::Result<()>> {
        self.project().inner.poll_read(cx, buf)
    }
}

impl<W: AsyncWrite, D: Duration> AsyncWrite for DelayedWriter<W, D> {
    fn poll_write(
        self: Pin<&mut Self>,
        cx: &mut Context<'_>,
        buf: &[u8],
    ) -> Poll<io::Result<usize>> {
        self.poll_with_write_delay(cx, |cx, inner| inner.poll_write(cx, buf))
    }

    fn poll_flush(self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<io::Result<()>> {
        self.project().inner.poll_flush(cx)
    }

    fn poll_shutdown(self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<io::Result<()>> {
        self.poll_with_write_delay(cx, |cx, inner| inner.poll_shutdown(cx))
    }

    fn is_write_vectored(&self) -> bool {
        self.inner.is_write_vectored()
    }

    fn poll_write_vectored(
        self: Pin<&mut Self>,
        cx: &mut Context<'_>,
        bufs: &[io::IoSlice<'_>],
    ) -> Poll<io::Result<usize>> {
        self.poll_with_write_delay(cx, |cx, inner| inner.poll_write_vectored(cx, bufs))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use tokio::io::{duplex, AsyncReadExt, AsyncWriteExt, BufReader};
    use tokio::time;

    #[test]
    fn duration_set_get_roundtrip() {
        let d = DynamicDuration::default();
        assert!(d.duration().is_zero());

        d.set(time::Duration::from_millis(150));
        assert_eq!(d.duration(), time::Duration::from_millis(150));
        assert!(!d.duration().is_zero());

        d.set(time::Duration::ZERO);
        assert_eq!(d.duration().as_nanos(), 0);
    }

    #[tokio::test(start_paused = true)]
    async fn delayed_writer_waits_before_each_write() {
        let (w, mut r) = duplex(64);
        let d = DynamicDuration::new(time::Duration::from_millis(100));
        let w = DelayedWriter::new(w, d.clone());

        let write = tokio::spawn(async move {
            let mut dw2 = w;
            dw2.write_all(b"hi").await.unwrap();
            dw2.flush().await.unwrap();
        });

        let try_read = tokio::time::timeout(time::Duration::from_millis(1), r.read_u8()).await;
        assert!(try_read.is_err(), "unexpectedly read before delay elapsed");

        time::sleep(time::Duration::from_millis(100)).await;

        let mut buf = [0u8; 2];
        r.read_exact(&mut buf).await.unwrap();
        assert_eq!(&buf, b"hi");

        write.await.unwrap();
    }

    #[tokio::test(start_paused = true)]
    async fn delayed_reader_only_on_new_bursts() {
        let (mut w, r) = duplex(64);
        let br = BufReader::new(r);
        let d = DynamicDuration::new(time::Duration::from_millis(50));
        let mut dr = DelayedReader::new(br, d.clone());

        tokio::spawn(async move {
            let _ = w.write_all(b"abcdef").await;
        });

        let mut buf = [0u8; 3];
        let pending =
            tokio::time::timeout(time::Duration::from_millis(1), dr.read_exact(&mut buf)).await;
        assert!(pending.is_err());

        time::sleep(time::Duration::from_millis(50)).await;
        dr.read_exact(&mut buf).await.unwrap();
        assert_eq!(&buf, b"abc");

        let mut buf2 = [0u8; 3];
        dr.read_exact(&mut buf2).await.unwrap();
        assert_eq!(&buf2, b"def");

        let (mut w2, r2) = duplex(64);
        let br2 = BufReader::new(r2);
        let mut dr2 = DelayedReader::new(br2, d.clone());

        tokio::spawn(async move {
            let _ = w2.write_all(b"ZZ").await;
        });

        let mut buf3 = [0u8; 2];
        let pending2 =
            tokio::time::timeout(time::Duration::from_millis(1), dr2.read_exact(&mut buf3)).await;
        assert!(pending2.is_err());
        time::sleep(time::Duration::from_millis(50)).await;
        dr2.read_exact(&mut buf3).await.unwrap();
        assert_eq!(&buf3, b"ZZ");
    }

    #[tokio::test(start_paused = true)]
    async fn delayed_writer_zero_delay_fast_path() {
        let (w, mut r) = duplex(64);
        let d = DynamicDuration::new(time::Duration::ZERO);
        let mut dw = DelayedWriter::new(w, d);

        dw.write_all(b"ok").await.unwrap();
        dw.flush().await.unwrap();
        let mut buf = [0u8; 2];
        r.read_exact(&mut buf).await.unwrap();
        assert_eq!(&buf, b"ok");
    }

    #[tokio::test(start_paused = true)]
    async fn delayed_writer_shutdown_is_delayed() {
        let (w, _r) = duplex(64);
        let d = DynamicDuration::new(time::Duration::from_millis(40));
        let mut dw = DelayedWriter::new(w, d);

        let mut shutdown = Box::pin(futures::future::poll_fn(|cx| {
            Pin::new(&mut dw).poll_shutdown(cx)
        }));

        let early = tokio::time::timeout(time::Duration::from_millis(1), &mut shutdown).await;
        assert!(early.is_err(), "shutdown completed before delay");

        time::sleep(time::Duration::from_millis(40)).await;
        shutdown.await.unwrap();
    }

    #[tokio::test(start_paused = true)]
    async fn delayed_writer_vectored_write_is_delayed() {
        use futures::future::poll_fn;
        use std::io::IoSlice;

        let (w, mut r) = duplex(64);
        let d = DynamicDuration::new(time::Duration::from_millis(25));
        let mut dw = DelayedWriter::new(w, d);

        let a = IoSlice::new(b"hello ");
        let b = IoSlice::new(b"vectored");

        let mut fut = Box::pin(poll_fn(|cx| {
            Pin::new(&mut dw).poll_write_vectored(cx, &[a, b])
        }));

        let early = tokio::time::timeout(time::Duration::from_millis(1), &mut fut).await;
        assert!(early.is_err(), "vectored write completed before delay");

        time::sleep(time::Duration::from_millis(25)).await;
        let n = fut.await.unwrap();
        assert_eq!(n, b"hello vectored".len());

        let mut buf = vec![0u8; n];
        r.read_exact(&mut buf).await.unwrap();
        assert_eq!(&buf, b"hello vectored");
    }

    #[tokio::test(start_paused = true)]
    async fn delayed_writer_dynamic_toggle_runtime() {
        let (dw, mut r) = duplex(64);
        let d = DynamicDuration::new(time::Duration::from_millis(30));
        let mut dw = DelayedWriter::new(dw, d.clone());

        let write1 = tokio::spawn(async move {
            dw.write_all(b"A").await.unwrap();
            dw.flush().await.unwrap();
            dw
        });

        let early = tokio::time::timeout(time::Duration::from_millis(1), r.read_u8()).await;
        assert!(early.is_err());
        time::sleep(time::Duration::from_millis(30)).await;
        assert_eq!(r.read_u8().await.unwrap(), b'A');
        let mut dw = write1.await.unwrap();

        d.set(time::Duration::ZERO);
        dw.write_all(b"B").await.unwrap();
        dw.flush().await.unwrap();
        assert_eq!(r.read_u8().await.unwrap(), b'B');
    }

    #[tokio::test(start_paused = true)]
    async fn delayed_writer_flush_is_not_delayed() {
        let (w, _r) = duplex(64);
        let d = DynamicDuration::new(time::Duration::from_millis(50));
        let mut dw = DelayedWriter::new(w, d);

        tokio::time::timeout(time::Duration::from_millis(1), dw.flush())
            .await
            .expect("flush should not be delayed")
            .unwrap();
    }

    #[tokio::test(start_paused = true)]
    async fn delayed_reader_zero_delay_fast_path() {
        let (mut w, r) = duplex(64);
        let br = BufReader::new(r);
        let d = DynamicDuration::new(time::Duration::ZERO);
        let mut dr = DelayedReader::new(br, d);

        w.write_all(b"OK").await.unwrap();
        w.flush().await.unwrap();

        let mut buf = [0u8; 2];
        tokio::time::timeout(time::Duration::from_millis(1), dr.read_exact(&mut buf))
            .await
            .expect("read should be immediate")
            .unwrap();
        assert_eq!(&buf, b"OK");
    }

    #[tokio::test(start_paused = true)]
    async fn delayed_reader_dynamic_toggle_runtime() {
        let (mut w, r) = duplex(64);
        let br = BufReader::new(r);
        let d = DynamicDuration::new(time::Duration::ZERO);
        let mut dr = DelayedReader::new(br, d.clone());

        w.write_all(b"11").await.unwrap();
        w.flush().await.unwrap();
        let mut buf = [0u8; 2];
        tokio::time::timeout(time::Duration::from_millis(1), dr.read_exact(&mut buf))
            .await
            .expect("read should be immediate")
            .unwrap();
        assert_eq!(&buf, b"11");

        d.set(time::Duration::from_millis(40));
        w.write_all(b"22").await.unwrap();
        w.flush().await.unwrap();

        let pending =
            tokio::time::timeout(time::Duration::from_millis(1), dr.read_exact(&mut buf)).await;
        assert!(pending.is_err(), "read completed before delay");
        time::sleep(time::Duration::from_millis(40)).await;
        dr.read_exact(&mut buf).await.unwrap();
        assert_eq!(&buf, b"22");
    }

    #[tokio::test(start_paused = true)]
    async fn delayed_reader_forwards_writes_without_delay() {

        let (w, mut r) = duplex(64);
        let d = DynamicDuration::new(time::Duration::from_millis(100));
        let mut drw = DelayedReader::new(w, d);

        drw.write_all(b"pw").await.unwrap();
        drw.flush().await.unwrap();

        let mut buf = [0u8; 2];
        tokio::time::timeout(time::Duration::from_millis(1), r.read_exact(&mut buf))
            .await
            .expect("write path should not be delayed")
            .unwrap();
        assert_eq!(&buf, b"pw");
    }
}
