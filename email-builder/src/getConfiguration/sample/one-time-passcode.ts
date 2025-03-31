import { TEditorConfiguration } from '../../documents/editor/core';

const ONE_TIME_PASSCODE: TEditorConfiguration = {
  root: {
    type: 'EmailLayout',
    data: {
      backdropColor: '#000000',
      canvasColor: '#000000',
      textColor: '#FFFFFF',
      fontFamily: 'BOOK_SERIF',
      childrenIds: [
        'block_ChPX66qUhF46uynDE8AY11',
        'block_CkNrtQgkqPt2YWLv1hr5eJ',
        'block_BFLBa3q5y8kax9KngyXP65',
        'block_4T7sDFb4rqbSyWjLGJKmov',
        'block_Rvc8ZfTjfhXjpphHquJKvP',
      ],
    },
  },
  block_ChPX66qUhF46uynDE8AY11: {
    type: 'Image',
    data: {
      style: {
        backgroundColor: null,
        padding: {
          top: 24,
          bottom: 24,
          left: 24,
          right: 24,
        },
        textAlign: 'center',
      },
      props: {
        height: 24,
        url: 'https://d1iiu589g39o6c.cloudfront.net/live/platforms/platform_A9wwKSL6EV6orh6f/images/wptemplateimage_jc7ZfPvdHJ6rtH1W/&.png',
        contentAlignment: 'middle',
      },
    },
  },
  block_CkNrtQgkqPt2YWLv1hr5eJ: {
    type: 'Text',
    data: {
      style: {
        color: '#ffffff',
        backgroundColor: null,
        fontSize: 16,
        fontFamily: null,
        fontWeight: 'normal',
        textAlign: 'center',
        padding: {
          top: 16,
          bottom: 16,
          left: 24,
          right: 24,
        },
      },
      props: {
        text: 'Here is your one-time passcode:',
      },
    },
  },
  block_BFLBa3q5y8kax9KngyXP65: {
    type: 'Heading',
    data: {
      style: {
        color: null,
        backgroundColor: null,
        fontFamily: 'MONOSPACE',
        fontWeight: 'bold',
        textAlign: 'center',
        padding: {
          top: 16,
          bottom: 16,
          left: 24,
          right: 24,
        },
      },
      props: {
        level: 'h1',
        text: '0123456',
      },
    },
  },
  block_4T7sDFb4rqbSyWjLGJKmov: {
    type: 'Text',
    data: {
      style: {
        color: '#868686',
        backgroundColor: null,
        fontSize: 16,
        fontFamily: null,
        fontWeight: 'normal',
        textAlign: 'center',
        padding: {
          top: 16,
          bottom: 16,
          left: 24,
          right: 24,
        },
      },
      props: {
        text: 'This code will expire in 30 minutes.',
      },
    },
  },
  block_Rvc8ZfTjfhXjpphHquJKvP: {
    type: 'Text',
    data: {
      style: {
        color: '#868686',
        backgroundColor: null,
        fontSize: 14,
        fontFamily: null,
        fontWeight: 'normal',
        textAlign: 'center',
        padding: {
          top: 16,
          bottom: 16,
          left: 24,
          right: 24,
        },
      },
      props: {
        text: 'Problems? Just reply to this email.',
      },
    },
  },
};

export default ONE_TIME_PASSCODE;
