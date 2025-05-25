use std::time::Duration;

use bytes::Bytes;
use tokio::{
    io::{self, AsyncReadExt, AsyncWriteExt},
    net::TcpStream,
    sync::{mpsc, oneshot},
};
use tokio_netem::{
    corrupter::Corrupter,
    injector::ReadInjector,
    io::{NetEmReadExt, NetEmWriteExt},
    shutdowner::Shutdowner,
    terminator::Terminator,
};

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let stream = TcpStream::connect("localhost:8080").await?;

    let (_inject_tx, inject_rx) = mpsc::channel(12);

    let stream = Terminator::new(stream, 0.1);
    let stream = Corrupter::new(stream, 0.1);
    let _stream = stream
        .inject_read(inject_rx)
        .slice_writes(12)
        .delay_writes(Duration::from_secs(1));

    todo!()
}

async fn _example1() -> io::Result<()> {

    let mut stream = TcpStream::connect("localhost:80")
        .await?
        .throttle_writes(32 * 1024)
        .slice_writes(16);
    stream.write_all(b"ping").await?;

    let mut stream = Terminator::new(stream, 0.01);

    let mut buf = [0u8; 4];
    stream.read_exact(&mut buf).await?;

    let (tx_err, rx_err) = oneshot::channel();
    let stream = Shutdowner::new(stream, rx_err);
    tx_err.send(io::Error::other("unexpected").into()).unwrap();

    let stream = Corrupter::new(stream, 0.005);

    let (tx_data, rx_data) = mpsc::channel(1);

    let mut _stream = ReadInjector::new(stream, rx_data);
    tx_data
        .send(Bytes::from_static(b"@*(@*!(#*!("))
        .await
        .unwrap();

    todo!()
}
