<?php

abstract class tsdb {
	protected static $host = "localhost";
	protected static $port = "4080";
	protected static $base = "api/v1/";

	public static function query( $series, $limit = 0, $offset = 0, $order = "desc" ) {
		$ch = self::getCurl( "query", array( "series" => $series, "limit" => $limit, "offset" => $offset, "order" => $order ) );
		$result = self::execCurl( $ch, true );
		if ( isset( $result["status"] ) && $result["status"] == "error" ) {
			throw new \RuntimeException( $result["message"] );
		}

		return $result["result"];
	}

	public static function write( $series, $value, $time = "", $ttl = "0s") {
		$ch = self::getCurl( "write", array( "series" => $series, "value" => $value, "time" => $time, "ttl" => $ttl ) );
		$result = self::execCurl( $ch, true );

		if ( isset( $result["status"] ) && $result["status"] == "error" ) {
			throw new \RuntimeException( $result["message"] );
		}

		return true;
	}

	public static function asyncwrite( $series, $value, $time = "", $ttl = "0s") {
		$ch = self::getCurl( "asyncwrite", array( "series" => $series, "value" => $value, "time" => $time, "ttl" => $ttl ) );
		$result = self::execCurl( $ch, true );

		if ( isset( $result["status"] ) && $result["status"] == "error" ) {
			throw new \RuntimeException( $result["message"] );
		}

		return true;
	}

	public static function delete( $series ) {
		$ch = self::getCurl( "delete", array( "series" => $series ) );
		$result = self::execCurl( $ch, true );

		if ( isset( $result["status"] ) && $result["status"] == "error" ) {
			throw new \RuntimeException( $result["message"] );
		}

		return true;
	}

	public static function deletebytime( $series, $time ) {
		$ch = self::getCurl( "deletebytime", array( "series" => $series, "time" => $time ) );
		$result = self::execCurl( $ch, true );

		if ( isset( $result["status"] ) && $result["status"] == "error" ) {
			throw new \RuntimeException( $result["message"] );
		}

		return true;
	}

	public static function count( $series ) {
		$ch = self::getCurl( "count", array( "series" => $series ) );
		$result = self::execCurl( $ch, true );

		if ( isset( $result["status"] ) && $result["status"] == "error" ) {
			throw new \RuntimeException( $result["message"] );
		}

		return $result["result"];
	}

	protected static function getCurl( $uri, array $args = array() ) {
		$url  = "http://".self::$host.":".self::$port."/".self::$base."".$uri;
		$url .= "?" . http_build_query( $args );
		$ch   = curl_init( $url );
		curl_setopt( $ch, CURLOPT_RETURNTRANSFER, true );
		curl_setopt( $ch, CURLOPT_CONNECTTIMEOUT, 4 );
		curl_setopt( $ch, CURLOPT_TIMEOUT, 10 );
		return $ch;
	}

	protected static function execCurl( $ch, $json = false ) {
		$response = curl_exec( $ch );
		$status   = (string)curl_getinfo( $ch, CURLINFO_HTTP_CODE );
		//$type     = curl_getinfo($ch, CURLINFO_CONTENT_TYPE);
		curl_close( $ch );
		if ( $status[0] != 2 ) {
			throw new \RuntimeException( $response );
		}
		return $json ? json_decode( $response, true ) : $response;
	}
}

?>
