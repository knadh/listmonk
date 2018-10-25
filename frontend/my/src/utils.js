import React from 'react'
import ReactDOM from 'react-dom';

import { Alert } from 'antd';


class Utils {
    static months = [ "Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec" ]
    static days = [ "Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat" ]

    // Converts the ISO date format to a simpler form.
    static DateString = (stamp, showTime) => {
        if(!stamp) {
            return ""
        }

        let d = new Date(stamp)

        let out = Utils.days[d.getDay()] + ", " + d.getDate() + " " + Utils.months[d.getMonth()] + " " + d.getFullYear()
        if(showTime) {
            out += " " + d.getHours() + ":" + d.getMinutes()
        }

        return out
    }

    // HttpError takes an axios error and returns an error dict after some sanity checks.
    static HttpError = (err) => {
        if (!err.response) {
            return err
        }
        
        if(!err.response.data || !err.response.data.message) {
            return {
                "message": err.message + " - " + err.response.request.responseURL,
                "data": {}
            }            
        }

        return {
            "message": err.response.data.message,
            "data": err.response.data.data
        }
    }

    // Shows a flash message.
    static Alert = (msg, msgType) => {
        document.getElementById('alert-container').classList.add('visible')
        ReactDOM.render(<Alert message={ msg } type={ msgType } showIcon />,
            document.getElementById('alert-container'))
    }
    static ModalAlert = (msg, msgType) => {
        document.getElementById('modal-alert-container').classList.add('visible')
        ReactDOM.render(<Alert message={ msg } type={ msgType } showIcon />,
            document.getElementById('modal-alert-container'))
    }
}

export default Utils
