use std::{
    error::Error,
    fmt, io,
    pin::Pin,
    task::{Context, Poll},
};

use pin_project::pin_project;
use rand::{rng, rngs::SmallRng, RngCore, SeedableRng};
use tokio::io::{AsyncBufRead, AsyncRead, AsyncWrite, ReadBuf};

use crate::{
    io::ResetLinger,
    probability::{try_trigger, Probability},
};

#[derive(Debug, Copy, Clone)]
pub struct TerminatedError;

pub const TERMINATED_ERROR: TerminatedError = TerminatedError;

impl fmt::Display for TerminatedError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "stream was terminated by tokio-netem terminator")
    }
}

impl Error for TerminatedError {}

#[pin_project]
pub struct Terminator<T, P> {
    #[pin]
    inner: T,
    prob: P,
    rng: SmallRng,
    triggered: bool,
}

impl<T, P: Probability> Terminator<T, P> {
    pub fn new(inner: T, prob: P) -> Self {
        Terminator {
            inner,
            prob,
            rng: SmallRng::from_rng(&mut rng()),
            triggered: false,
        }
    }

    pub fn from_seed(inner: T, prob: P, seed: [u8; 32]) -> Self {
        Terminator {
            inner,
            prob,
            rng: SmallRng::from_seed(seed),
            triggered: false,
        }
    }

    pub fn from_rng(inner: T, prob: P, rng: &mut impl RngCore) -> Self {
        Terminator {
            inner,
            prob,
            rng: SmallRng::from_rng(rng),
            triggered: false,
        }
    }
}

impl<T: ResetLinger, P> ResetLinger for Terminator<T, P> {
    fn set_reset_linger(&mut self) -> io::Result<()> {
        self.inner.set_reset_linger()
    }
}

impl<T: AsyncBufRead, P: Probability> AsyncBufRead for Terminator<T, P> {
    fn poll_fill_buf(self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<io::Result<&[u8]>> {
        self.project().inner.poll_fill_buf(cx)
    }

    fn consume(self: Pin<&mut Self>, amt: usize) {
        self.project().inner.consume(amt)
    }
}

impl<R: AsyncRead, P: Probability> AsyncRead for Terminator<R, P> {
    fn poll_read(
        mut self: Pin<&mut Self>,
        cx: &mut Context<'_>,
        buf: &mut ReadBuf<'_>,
    ) -> Poll<io::Result<()>> {
        let this = self.as_mut().project();
        if try_trigger(this.triggered, this.rng, this.prob) {
            return Poll::Ready(Err(io::Error::other(TERMINATED_ERROR)));
        }

        this.inner.poll_read(cx, buf)
    }
}

impl<W: AsyncWrite, P: Probability> AsyncWrite for Terminator<W, P> {
    fn poll_write(
        mut self: Pin<&mut Self>,
        cx: &mut Context<'_>,
        buf: &[u8],
    ) -> Poll<io::Result<usize>> {
        let this = self.as_mut().project();
        if try_trigger(this.triggered, this.rng, this.prob) {
            return Poll::Ready(Err(io::Error::other(TERMINATED_ERROR)));
        }

        this.inner.poll_write(cx, buf)
    }

    fn poll_flush(mut self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<io::Result<()>> {
        let this = self.as_mut().project();
        if try_trigger(this.triggered, this.rng, this.prob) {
            return Poll::Ready(Err(io::Error::other(TERMINATED_ERROR)));
        }

        this.inner.poll_flush(cx)
    }

    fn poll_shutdown(mut self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<io::Result<()>> {
        let this = self.as_mut().project();
        if try_trigger(this.triggered, this.rng, this.prob) {
            return Poll::Ready(Err(io::Error::other(TERMINATED_ERROR)));
        }

        this.inner.poll_shutdown(cx)
    }

    fn is_write_vectored(&self) -> bool {
        self.inner.is_write_vectored()
    }

    fn poll_write_vectored(
        mut self: Pin<&mut Self>,
        cx: &mut Context<'_>,
        bufs: &[io::IoSlice<'_>],
    ) -> Poll<io::Result<usize>> {
        let this = self.as_mut().project();
        if try_trigger(this.triggered, this.rng, this.prob) {
            return Poll::Ready(Err(io::Error::other(TERMINATED_ERROR)));
        }

        this.inner.poll_write_vectored(cx, bufs)
    }
}

impl<RW: fmt::Debug, P> fmt::Debug for Terminator<RW, P> {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        self.inner.fmt(f)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::io::ResetLinger;
    use crate::probability::DynamicProbability;
    use pin_project::pin_project;
    use std::pin::Pin;
    use std::{
        io,
        task::{Context, Poll},
    };
    use tokio::io::{duplex, AsyncReadExt, AsyncWriteExt};

    #[tokio::test]
    async fn write_fails_immediately_with_prob_one_and_stays_failed() {
        let (mut w, _) = duplex(64);
        let mut term = Terminator::new(&mut w, 1.0f64);

        let err = term.write_all(b"boom").await.unwrap_err();
        assert_eq!(
            err.to_string(),
            io::Error::other(TERMINATED_ERROR).to_string()
        );

        let err2 = term.flush().await.unwrap_err();
        assert_eq!(
            err2.to_string(),
            io::Error::other(TERMINATED_ERROR).to_string()
        );

        let err3 = term.read_u8().await.unwrap_err();
        assert_eq!(
            err3.to_string(),
            io::Error::other(TERMINATED_ERROR).to_string()
        );
    }

    #[tokio::test]
    async fn read_and_write_pass_with_prob_zero() {
        let (mut w, mut r) = duplex(64);
        let mut tw = Terminator::new(&mut w, 0.0f64);
        let mut tr = Terminator::new(&mut r, 0.0f64);

        tw.write_all(b"hello").await.unwrap();
        tw.flush().await.unwrap();

        let mut buf = [0u8; 5];
        tr.read_exact(&mut buf).await.unwrap();
        assert_eq!(&buf, b"hello");
    }

    #[tokio::test]
    async fn vectored_write_path_covered() {
        let (mut w, mut r) = duplex(128);
        let mut term = Terminator::new(&mut w, 0.0f64);
        let a = io::IoSlice::new(b"hello ");
        let b = io::IoSlice::new(b"vectored");
        let n = term.write_vectored(&[a, b]).await.unwrap();
        term.flush().await.unwrap();

        let mut buf = vec![0u8; n];
        r.read_exact(&mut buf).await.unwrap();
        assert_eq!(&buf, &b"hello vectored"[..n]);
    }

    #[tokio::test]
    async fn dynamic_probability_runtime_update() {
        let (mut w, _r) = duplex(64);
        let prob = DynamicProbability::new(0.0).unwrap();
        let mut term = Terminator::new(&mut w, prob.clone());
        term.write_all(b"x").await.unwrap();
        prob.set(1.0).unwrap();
        let err = term.write_all(b"y").await.unwrap_err();
        assert_eq!(
            err.to_string(),
            io::Error::other(TERMINATED_ERROR).to_string()
        );
    }

    #[test]
    fn invalid_probability_rejected() {
        let err = DynamicProbability::new(1.5).unwrap_err();
        assert_eq!(err.kind(), io::ErrorKind::InvalidInput);
    }

    #[pin_project]
    struct Stub {
        reset_called: bool,
    }

    impl Stub {
        fn new() -> Self {
            Self {
                reset_called: false,
            }
        }
    }

    impl AsyncWrite for Stub {
        fn poll_write(
            self: Pin<&mut Self>,
            _cx: &mut Context<'_>,
            _buf: &[u8],
        ) -> Poll<io::Result<usize>> {
            Poll::Ready(Ok(0))
        }
        fn poll_flush(self: Pin<&mut Self>, _cx: &mut Context<'_>) -> Poll<io::Result<()>> {
            Poll::Ready(Ok(()))
        }
        fn poll_shutdown(self: Pin<&mut Self>, _cx: &mut Context<'_>) -> Poll<io::Result<()>> {
            Poll::Ready(Ok(()))
        }
    }

    impl ResetLinger for Stub {
        fn set_reset_linger(&mut self) -> io::Result<()> {
            self.reset_called = true;
            Ok(())
        }
    }

    #[test]
    fn reset_linger_forwards() {
        let s = Stub::new();
        let mut t = Terminator::new(s, 0.0f64);
        ResetLinger::set_reset_linger(&mut t).unwrap();
    }

    #[tokio::test]
    async fn write_vectored_fails_when_prob_one() {
        let (mut w, _r) = duplex(128);
        let mut term = Terminator::new(&mut w, 1.0f64);

        let a = io::IoSlice::new(b"hello ");
        let b = io::IoSlice::new(b"vectored");
        let err = term.write_vectored(&[a, b]).await.unwrap_err();
        assert_eq!(
            err.to_string(),
            io::Error::other(TERMINATED_ERROR).to_string()
        );
    }

    #[tokio::test]
    async fn read_fails_immediately_with_prob_one() {
        let (_w, mut r) = duplex(64);
        let mut tr = Terminator::new(&mut r, 1.0f64);
        let err = tr.read_u8().await.unwrap_err();
        assert_eq!(
            err.to_string(),
            io::Error::other(TERMINATED_ERROR).to_string()
        );
    }

    #[tokio::test]
    async fn shutdown_fails_with_prob_one() {
        let (mut w, _r) = duplex(64);
        let mut t = Terminator::new(&mut w, 1.0f64);
        let err = t.shutdown().await.unwrap_err();
        assert_eq!(
            err.to_string(),
            io::Error::other(TERMINATED_ERROR).to_string()
        );
    }

    #[tokio::test]
    async fn f64_probability_impl_clamps_below_zero_and_above_one() {
        let (mut w1, mut r1) = duplex(64);
        let mut t1w = Terminator::new(&mut w1, -0.25f64);
        let mut t1r = Terminator::new(&mut r1, -0.25f64);
        t1w.write_all(b"ok").await.unwrap();
        t1w.flush().await.unwrap();
        let mut buf = [0u8; 2];
        t1r.read_exact(&mut buf).await.unwrap();
        assert_eq!(&buf, b"ok");

        let (mut w2, _r2) = duplex(64);
        let mut t2 = Terminator::new(&mut w2, 1.25f64);
        let err = t2.write_all(b"x").await.unwrap_err();
        assert_eq!(
            err.to_string(),
            io::Error::other(TERMINATED_ERROR).to_string()
        );
    }

    #[test]
    fn dynamic_probability_boundaries_accept() {
        let p0 = DynamicProbability::new(0.0).unwrap();
        assert_eq!(p0.probability(), 0.0);
        let p1 = DynamicProbability::new(1.0).unwrap();
        assert_eq!(p1.threshold(), u64::MAX);
    }

    #[test]
    fn invalid_probability_nan_and_range_rejected() {

        let e_hi = DynamicProbability::new(1.01).unwrap_err();
        assert_eq!(e_hi.kind(), io::ErrorKind::InvalidInput);

        let e_lo = DynamicProbability::new(-0.0001).unwrap_err();
        assert_eq!(e_lo.kind(), io::ErrorKind::InvalidInput);

        let nan = f64::NAN;
        let e_nan = DynamicProbability::new(nan).unwrap_err();
        assert_eq!(e_nan.kind(), io::ErrorKind::InvalidInput);
    }

    #[test]
    fn dynamic_probability_set_invalid_rejected_and_state_unchanged() {
        let prob = DynamicProbability::new(0.5).unwrap();
        let before_rate = prob.probability();
        let before_thr = prob.threshold();

        let err = prob.set(2.0).unwrap_err();
        assert_eq!(err.kind(), io::ErrorKind::InvalidInput);

        assert_eq!(prob.probability(), before_rate);
        assert_eq!(prob.threshold(), before_thr);
    }

    #[test]
    fn dynamic_probability_threshold_updates() {
        let prob = DynamicProbability::new(0.0).unwrap();
        assert_eq!(prob.threshold(), 0);

        prob.set(0.5).unwrap();
        let expected = (0.5f64 * u64::MAX as f64) as u64;
        assert_eq!(prob.threshold(), expected);

        prob.set(1.0).unwrap();
        assert_eq!(prob.threshold(), u64::MAX);
    }

    #[pin_project]
    struct VectoredStub {
        pub wrote: bool,
    }

    impl AsyncWrite for VectoredStub {
        fn poll_write(
            self: Pin<&mut Self>,
            _cx: &mut Context<'_>,
            _buf: &[u8],
        ) -> Poll<io::Result<usize>> {
            *self.project().wrote = true;
            Poll::Ready(Ok(0))
        }
        fn poll_flush(self: Pin<&mut Self>, _cx: &mut Context<'_>) -> Poll<io::Result<()>> {
            Poll::Ready(Ok(()))
        }
        fn poll_shutdown(self: Pin<&mut Self>, _cx: &mut Context<'_>) -> Poll<io::Result<()>> {
            Poll::Ready(Ok(()))
        }
        fn is_write_vectored(&self) -> bool {
            true
        }
        fn poll_write_vectored(
            self: Pin<&mut Self>,
            _cx: &mut Context<'_>,
            _bufs: &[io::IoSlice<'_>],
        ) -> Poll<io::Result<usize>> {
            Poll::Ready(Ok(0))
        }
    }

    #[test]
    fn is_write_vectored_is_forwarded() {
        let stub = VectoredStub { wrote: false };
        let t = Terminator::new(stub, 0.0f64);
        assert!(t.is_write_vectored());
    }
}
