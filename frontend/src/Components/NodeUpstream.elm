module Components.NodeUpstream exposing (view)

import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Element exposing (..)
import Element.Border as Border
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style exposing (colors)
import UI.ViewIf exposing (viewIf)


view : NodeOptions msg -> Element msg
view o =
    let
        opts =
            oToInputO o 100

        textInput =
            NodeInputs.nodeTextInput opts ""

        checkboxInput =
            NodeInputs.nodeCheckboxInput opts ""

        disabled =
            Point.getBool o.node.points Point.typeDisable ""
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10 ]
            [ Icon.uploadCloud
            , text <|
                Point.getText o.node.points Point.typeDescription ""
            , viewIf disabled <| text "(disabled)"
            ]
            :: (if o.expDetail then
                    [ textInput Point.typeDescription "Description" ""
                    , textInput Point.typeURI "URI" "nats://myserver:4222, ws://myserver"
                    , textInput Point.typeAuthToken "Auth Token" ""
                    , checkboxInput Point.typeDisable "Disable"
                    ]

                else
                    []
               )
