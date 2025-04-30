use std::time::Duration;
use tokio::io::{self, AsyncReadExt, AsyncWriteExt, BufReader};
use tokio::net::TcpStream;
use tokio_netem::io::{NetEmReadExt, NetEmWriteExt};

#[tokio::main]
    async fn main() -> io::Result<()> {
    let stream = TcpStream::connect("127.0.0.1:12345").await?;
    let (reader, writer) = stream.into_split();

    let mut reader = BufReader::new(reader).throttle_reads(32);
    let mut writer = writer
        .delay_writes(Duration::from_millis(25))
        .slice_writes(8);

    writer.write_all(b"ping").await?;
    writer.flush().await?; // assumes the peer echoes data back
    let mut buf = [0u8; 4];
    reader.read_exact(&mut buf).await?;
    Ok(()) 
}

use std::sync::Arc;
// use std::time::Duration;
use tokio::io::{self, AsyncReadExt, AsyncWriteExt, BufReader};
use tokio::net::TcpStream;
use tokio_netem::io::{NetEmReadExt, NetEmWriteExt};
use tokio_netem::{
    terminator::Terminator,
    delayer::DynamicDuration,
    probability::DynamicProbability,
    throttler::DynamicRate,
};

#[tokio::main]
    async fn main() -> io::Result<()> {
    let rate: Arc<DynamicRate> = DynamicRate::new(0);
    let delay: Arc<DynamicDuration> = DynamicDuration::new(Duration::from_millis(0));
    let probability: Arc<DynamicProbability> = DynamicProbability::new(0.0)?;
    let stream = TcpStream::connect("127.0.0.1:12345").await?;
    let (reader_half, writer_half) = stream.into_split();

    let mut reader = BufReader::new(reader_half).throttle_reads_dyn(rate.clone());
    let writer_half = Terminator::new(writer_half, probability.clone());
    let mut writer = writer_half
        .delay_writes_dyn(delay.clone());

    rate.set(16);                       // tweak knobs at runtime
    delay.set(Duration::from_millis(50));
    probability.set(0.25)?;             // 25% chance to trip after this point
    writer.write_all(b"pong").await?;
    let mut buf = [0u8; 4];
    reader.read_exact(&mut buf).await?; // assumes the remote peer responds
    Ok(()) 
}