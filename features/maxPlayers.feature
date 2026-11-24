Feature: Lobbies can have a maximum number of players

  Background:
    Given the "signaling" backend is running


  Scenario: Players can set a maximum number of players for a lobby
    Given "blue" is connected as "1u8fw4aph5ypt" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" is connected as "h5yzwyizlwao" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "green" is connected as "19yrzmetd2bn7" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"

    And "blue" creates a lobby with these settings:
      """json
      {
        "maxPlayers": 2
      }
      """
    And "blue" receives the network event "lobby" with the argument "3t3cfgcqup9e"

    When "yellow" connects to the lobby "3t3cfgcqup9e"
    Then "yellow" receives the network event "lobby" with the argument "3t3cfgcqup9e"

    # The lobby is full
    When "green" tries to connect to the lobby "3t3cfgcqup9e" without a password
    Then "green" failed to join the lobby
    And the latest error for "green" is "lobby is full"

    When "blue" disconnects
    Then "blue" receives the network event "close"
    And "yellow" becomes the leader of the lobby

    # The lobby is not full anymore
    When "green" connects to the lobby "3t3cfgcqup9e"
    Then "green" receives the network event "lobby" with the argument "3t3cfgcqup9e"


  Scenario Outline: A member whose connection closed holds its seat only while it may be reconnecting
    Given the "testproxy" backend is running
    And "green" is connected as "1u8fw4aph5ypt" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And these peers exist:
      | peer     | game                                 | disconnected | last_seen            | updated_at           |
      | live     | 4307bd86-e1df-41b8-b9df-e22afcf084bd | false        | now                  | now                  |
      | live2    | 4307bd86-e1df-41b8-b9df-e22afcf084bd | false        | now                  | now                  |
      | dropping | 4307bd86-e1df-41b8-b9df-e22afcf084bd | true         | now                  | now                  |
      | crashed  | 4307bd86-e1df-41b8-b9df-e22afcf084bd | true         | 2000-01-01T00:00:00Z | 2000-01-01T00:00:00Z |
      | expired  | 4307bd86-e1df-41b8-b9df-e22afcf084bd | false        | 2000-01-01T00:00:00Z | 2000-01-01T00:00:00Z |
    And these lobbies exist:
      | code          | game                                 | peers   | public | max_players |
      | 0qva9vyurwbbl | 4307bd86-e1df-41b8-b9df-e22afcf084bd | <peers> | true   | 2           |

    When "green" tries to connect to the lobby "0qva9vyurwbbl" without a password
    Then the latest error for "green" is "<error>"

    Examples:
      | peers           | error         |
      | {live,live2}    | lobby is full |
      | {live,dropping} | lobby is full |

  Scenario Outline: A member gone for good frees its seat, to a code or a quick join
    Given the "testproxy" backend is running
    And "green" is connected as "1u8fw4aph5ypt" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And these peers exist:
      | peer    | game                                 | disconnected | last_seen            | updated_at           |
      | live    | 4307bd86-e1df-41b8-b9df-e22afcf084bd | false        | now                  | now                  |
      | crashed | 4307bd86-e1df-41b8-b9df-e22afcf084bd | true         | 2000-01-01T00:00:00Z | 2000-01-01T00:00:00Z |
      | expired | 4307bd86-e1df-41b8-b9df-e22afcf084bd | false        | 2000-01-01T00:00:00Z | 2000-01-01T00:00:00Z |
    And these lobbies exist:
      | code          | game                                 | peers   | public | max_players |
      | 0qva9vyurwbbl | 4307bd86-e1df-41b8-b9df-e22afcf084bd | <peers> | true   | 2           |

    When "green" <joins>
    Then "green" receives the network event "lobby" with the argument "0qva9vyurwbbl"

    Examples:
      | peers          | joins                                 |
      | {live,crashed} | connects to the lobby "0qva9vyurwbbl" |
      | {live,expired} | connects to the lobby "0qva9vyurwbbl" |

  Scenario: A quick join finds the seat a member gone for good left
    Given the "testproxy" backend is running
    And "green" is connected as "1u8fw4aph5ypt" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And these peers exist:
      | peer    | game                                 | disconnected | last_seen            | updated_at           |
      | live    | 4307bd86-e1df-41b8-b9df-e22afcf084bd | false        | now                  | now                  |
      | crashed | 4307bd86-e1df-41b8-b9df-e22afcf084bd | true         | 2000-01-01T00:00:00Z | 2000-01-01T00:00:00Z |
    And these lobbies exist:
      | code          | game                                 | peers          | public | max_players |
      | 0qva9vyurwbbl | 4307bd86-e1df-41b8-b9df-e22afcf084bd | {live,crashed} | true   | 2           |

    When "green" joins or creates a lobby with these settings:
      """json
      {}
      """
    Then "green" receives the network event "lobby" with the argument "0qva9vyurwbbl"

  Scenario: You can update the maximum number of players for a lobby
    Given "yellow" is connected as "1u8fw4aph5ypt" and ready for game "4307bd86-e1df-41b8-b9df-e22afcf084bd"
    And "yellow" creates a lobby with these settings:
      """json
      {
        "public": true
      }
      """
    And "yellow" receives the network event "lobby" with the argument "h5yzwyizlwao"

    When "yellow" requests lobbies with:
      """json
      {}
      """
    Then "yellow" should have received only these lobbies:
      | code         | maxPlayers |
      | h5yzwyizlwao | 4          |

    When "yellow" updates the lobby with these settings:
      """json
      {
        "maxPlayers": 12
      }
      """

    When "yellow" requests lobbies with:
      """json
      {}
      """
    Then "yellow" should have received only these lobbies:
      | code         | maxPlayers |
      | h5yzwyizlwao | 12         |
