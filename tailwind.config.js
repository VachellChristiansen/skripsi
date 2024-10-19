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
      },
    },
    plugins: [],
  }
  
  