import Head from 'next/head'

export default function Home() {
  const sendRequest = async (api) => {
    const req = await fetch(api, { method: 'POST' });
    const data = await req.json();

    return { data: data.results };
  };
  const sendOn = (event) => {
    event.preventDefault();

    sendRequest("/api/on");
  };
  const sendOff = (event) => {
    event.preventDefault();

    sendRequest("/api/off");
  };
  const sendFTB = (event) => {
    event.preventDefault();

    sendRequest("/api/ftb");
  };
  const sendDSK = (event) => {
    event.preventDefault();

    sendRequest("/api/dsk");
  };
  return (
    <div className="container">
      <Head>
        <title>VBS Lighting Bridge</title>
        <link rel="icon" href="/favicon.ico" />
      </Head>

      <main>
        <h1 className="title">
          <a href="https://github.com/kindlyops/vbs">VBS</a> Lighting bridge
        </h1>

        <div className="grid">
          <a href="#on" className="card" onClick={sendOn}>
            <h3>On &#128161;</h3>
            <p>light on.</p>
          </a>

          <a href="#off" className="card" onClick={sendOff}>
            <h3>Off &#128268;</h3>
            <p>light off.</p>
          </a>
          <a href="#ftb" className="card" onClick={sendFTB}>
            <h3>FTB &#10084;</h3>
            <p>fade to black</p>
          </a>
                    <a href="#off" className="card" onClick={sendDSK}>
            <h3>DSK &#129668;</h3>
            <p>toggle DSK.</p>
          </a>
        </div>
        <p className="description">
          Send <code>OSC</code> commands to Companion.app
        </p>
      </main>

      <footer>
        <a
          href="https://kindlyops.com?utm_source=vbs&utm_campaign=lighting-bridge"
          target="_blank"
          rel="noopener noreferrer"
        >
          Powered by{' '}
          <img src="/kindly-logo-small.png" alt="Kindly Ops" className="logo" />
        </a>
      </footer>

      <style jsx>{`
        .container {
          min-height: 100vh;
          padding: 0 0.5rem;
          display: flex;
          flex-direction: column;
          justify-content: center;
          align-items: center;
        }

        main {
          padding: 5rem 0;
          flex: 1;
          display: flex;
          flex-direction: column;
          justify-content: center;
          align-items: center;
        }

        footer {
          width: 100%;
          height: 100px;
          border-top: 1px solid #eaeaea;
          display: flex;
          justify-content: center;
          align-items: center;
        }

        footer img {
          margin-left: 0.5rem;
        }

        footer a {
          display: flex;
          justify-content: center;
          align-items: center;
        }

        a {
          color: inherit;
          text-decoration: none;
        }

        .title a {
          color: #0070f3;
          text-decoration: none;
        }

        .title a:hover,
        .title a:focus,
        .title a:active {
          text-decoration: underline;
        }

        .title {
          margin: 0;
          line-height: 1.15;
          font-size: 4rem;
        }

        .title,
        .description {
          text-align: center;
        }

        .description {
          line-height: 1.5;
          font-size: 1.5rem;
        }

        code {
          background: #fafafa;
          border-radius: 5px;
          padding: 0.75rem;
          font-size: 1.1rem;
          font-family: Menlo, Monaco, Lucida Console, Liberation Mono,
            DejaVu Sans Mono, Bitstream Vera Sans Mono, Courier New, monospace;
        }

        .grid {
          display: flex;
          align-items: center;
          justify-content: center;
          flex-direction: row;
          flex-wrap: wrap;

          max-width: 800px;
          margin-top: 3rem;
        }

        .card {
          margin: 1rem;
          flex-basis: 45%;
          padding: 1.5rem;
          text-align: left;
          color: inherit;
          text-decoration: none;
          border: 1px solid #eaeaea;
          border-radius: 10px;
          transition: color 0.15s ease, border-color 0.15s ease;
        }

        .card:hover,
        .card:focus,
        .card:active {
          color: #0070f3;
          border-color: #0070f3;
        }

        .card h3 {
          margin: 0 0 1rem 0;
          font-size: 1.5rem;
        }

        .card p {
          margin: 0;
          font-size: 1.25rem;
          line-height: 1.5;
        }

        .logo {
          height: 4em;
        }

        @media (max-width: 600px) {
          .grid {
            width: 100%;
            flex-direction: column;
          }
        }
      `}</style>

      <style jsx global>{`
        html,
        body {
          padding: 0;
          margin: 0;
          font-family: -apple-system, BlinkMacSystemFont, Segoe UI, Roboto,
            Oxygen, Ubuntu, Cantarell, Fira Sans, Droid Sans, Helvetica Neue,
            sans-serif;
        }

        * {
          box-sizing: border-box;
        }
      `}</style>
    </div>
  )
}
