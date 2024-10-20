/** @type {import('tailwindcss').Config} */
module.exports = {
    content: [
      "./web_views/*.{html,js}",
      "./web_views/components/*.{html,js}",
    ],
    theme: {
      extend: {
        fontFamily: {
          jsans: ["Josefin Sans", "sans-serif"],
          montserrat: ["Montserrat", "sans-serif"],
        },
        keyframes: {
          fadeToTransparentA: {
            '0%': { backgroundColor: 'rgb(254, 205, 211)', color: 'currentColor' },
            '100%': { backgroundColor: 'transparent', color: 'transparent' },
          },
          fadeToTransparentB: {
            '0%': { backgroundColor: 'rgb(167, 243, 208)', color: 'currentColor' },
            '100%': { backgroundColor: 'transparent', color: 'transparent' },
          },
        },
        animation: {
          fadeToTransparentA: 'fadeToTransparentA 1s forwards 5s',
          fadeToTransparentB: 'fadeToTransparentB 1s forwards 5s',
        },
      },
    },
    plugins: [],
  }
  
  