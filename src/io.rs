use std::{io, sync::Arc, time};

use bytes::Bytes;
use tokio::{
    io::{
        AsyncBufRead, AsyncRead, AsyncWrite, BufReader, BufStream, BufWriter, DuplexStream,
        SimplexStream,
    },
    sync::mpsc,
};

use crate::{
    delayer::{self, DynamicDuration},
    injector,
    slicer::{self, DynamicSize},
    throttler::{self, DynamicRate},
};

pub trait ResetLinger {
    fn set_reset_linger(&mut self) -> io::Result<()>;
}

impl ResetLinger for tokio::net::TcpStream {
    fn set_reset_linger(&mut self) -> io::Result<()> {
        tokio::net::TcpStream::set_linger(self, Some(time::Duration::from_secs(0)))
    }
}

impl ResetLinger for tokio::net::UnixStream {
    fn set_reset_linger(&mut self) -> io::Result<()> {
        Ok(())
    }
}

impl ResetLinger for tokio::net::unix::pipe::Receiver {
    fn set_reset_linger(&mut self) -> io::Result<()> {
        Ok(())
    }
}

impl ResetLinger for tokio::net::UdpSocket {
    fn set_reset_linger(&mut self) -> io::Result<()> {
        Ok(())
    }
}

impl<RW> ResetLinger for BufStream<RW> {
    fn set_reset_linger(&mut self) -> io::Result<()> {
        Ok(())
    }
}

impl<R: AsyncRead> ResetLinger for BufReader<R> {
    fn set_reset_linger(&mut self) -> io::Result<()> {
        Ok(())
    }
}

impl<W: AsyncWrite> ResetLinger for BufWriter<W> {
    fn set_reset_linger(&mut self) -> io::Result<()> {
        Ok(())
    }
}

impl ResetLinger for DuplexStream {
    fn set_reset_linger(&mut self) -> io::Result<()> {
        Ok(())
    }
}

impl ResetLinger for SimplexStream {
    fn set_reset_linger(&mut self) -> io::Result<()> {
        Ok(())
    }
}

pub trait NetEmReadExt: AsyncRead {
    #[must_use]
    fn throttle_reads(self, rate: usize) -> throttler::ThrottledReader<Self, usize>
    where
        Self: AsyncBufRead + Sized,
    {
        throttler::ThrottledReader::new(self, rate)
    }

    #[must_use]
    fn throttle_reads_dyn(
        self,
        rate: Arc<DynamicRate>,
    ) -> throttler::ThrottledReader<Self, Arc<DynamicRate>>
    where
        Self: AsyncBufRead + Sized,
    {
        throttler::ThrottledReader::new(self, rate)
    }


    #[must_use]
    fn delay_reads(self, delay: time::Duration) -> delayer::DelayedReader<Self, time::Duration>
    where
        Self: AsyncBufRead + Sized,
    {
        delayer::DelayedReader::new(self, delay)
    }

    #[must_use]
    fn delay_reads_dyn(
        self,
        delay: Arc<DynamicDuration>,
    ) -> delayer::DelayedReader<Self, Arc<DynamicDuration>>
    where
        Self: AsyncBufRead + Sized,
    {
        delayer::DelayedReader::new(self, delay)
    }

    #[must_use]
    fn inject_read(self, inject_rx: mpsc::Receiver<Bytes>) -> injector::ReadInjector<Self>
    where
        Self: Sized,
    {
        injector::ReadInjector::new(self, inject_rx)
    }
}

impl<T: AsyncRead> NetEmReadExt for T {}

pub trait NetEmWriteExt: AsyncWrite {

    #[must_use]
    fn throttle_writes(self, rate: usize) -> throttler::ThrottledWriter<Self, usize>
    where
        Self: Sized,
    {
        throttler::ThrottledWriter::new(self, rate)
    }

    #[must_use]
    fn throttle_writes_dyn(
        self,
        rate: Arc<DynamicRate>,
    ) -> throttler::ThrottledWriter<Self, Arc<DynamicRate>>
    where
        Self: Sized,
    {
        throttler::ThrottledWriter::new(self, rate)
    }

    #[must_use]
    fn delay_writes(self, delay: time::Duration) -> delayer::DelayedWriter<Self, time::Duration>
    where
        Self: Sized,
    {
        delayer::DelayedWriter::new(self, delay)
    }

    #[must_use]
    fn delay_writes_dyn(
        self,
        delay: Arc<delayer::DynamicDuration>,
    ) -> delayer::DelayedWriter<Self, Arc<DynamicDuration>>
    where
        Self: Sized,
    {
        delayer::DelayedWriter::new(self, delay)
    }

    #[must_use]
    fn slice_writes(self, size: usize) -> slicer::SlicedWriter<Self, usize>
    where
        Self: Sized,
    {
        slicer::SlicedWriter::new(self, size)
    }

    #[must_use]
    fn slice_writes_dyn(
        self,
        size: Arc<DynamicSize>,
    ) -> slicer::SlicedWriter<Self, Arc<DynamicSize>>
    where
        Self: Sized,
    {
        slicer::SlicedWriter::new(self, size)
    }

    #[must_use]
    fn inject_write(self, inject_rx: mpsc::Receiver<Bytes>) -> injector::WriteInjector<Self>
    where
        Self: Sized,
    {
        injector::WriteInjector::new(self, inject_rx)
    }
}

impl<T: AsyncWrite> NetEmWriteExt for T {}

#[cfg(test)]
mod tests {
    use super::*;
    use tokio::io::{duplex, AsyncReadExt, AsyncWriteExt};
    use tokio::time;

    #[tokio::test(start_paused = true)]
    async fn ext_throttled_write_and_slice_and_delay() {
        let (w, mut r) = duplex(256);
        let rate = 16usize;
        let size = 4usize;
        let delay = time::Duration::from_millis(10);

        let mut w = w
            .throttle_writes(rate)
            .slice_writes(size)
            .delay_writes(delay);

        w.write_all(b"hello world").await.unwrap();
        w.flush().await.unwrap();

        let mut buf = vec![0u8; 11];
        r.read_exact(&mut buf).await.unwrap();
        assert_eq!(&buf, b"hello world");
    }
}